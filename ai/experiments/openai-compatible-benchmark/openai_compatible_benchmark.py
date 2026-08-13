#!/usr/bin/env python3
"""Stdlib benchmark client for an OpenAI-compatible chat-completions endpoint.

The client deliberately records raw request measurements as JSONL. It does not
claim that first SSE event is token-level TTFT or that byte equality proves a
server-side prefix-cache hit.
"""

from __future__ import annotations

import argparse
import concurrent.futures
import json
import math
import statistics
import sys
import time
import urllib.error
import urllib.request
from collections.abc import Iterable
from dataclasses import asdict, dataclass


@dataclass
class Record:
    request_id: int
    ttft_ms: float | None
    e2e_ms: float
    completion_tokens: int | None
    approx_tpot_ms: float | None
    error: str | None = None


def percentile(values: list[float], fraction: float) -> float | None:
    if not values:
        return None
    ordered = sorted(values)
    index = max(0, math.ceil(fraction * len(ordered)) - 1)
    return ordered[index]


def summary(records: Iterable[Record], elapsed_seconds: float) -> dict[str, float | int | None]:
    values = list(records)
    successful = [record for record in values if record.error is None]
    ttft = [record.ttft_ms for record in successful if record.ttft_ms is not None]
    tpot = [record.approx_tpot_ms for record in successful if record.approx_tpot_ms is not None]
    completion_tokens = [record.completion_tokens for record in successful if record.completion_tokens is not None]
    return {
        "requests": len(values),
        "successful_requests": len(successful),
        "failed_requests": len(values) - len(successful),
        "ttft_p50_ms": percentile(ttft, 0.50),
        "ttft_p95_ms": percentile(ttft, 0.95),
        "e2e_p50_ms": percentile([record.e2e_ms for record in successful], 0.50),
        "approx_tpot_p50_ms": percentile(tpot, 0.50),
        "completion_tokens": sum(completion_tokens) if completion_tokens else None,
        "output_tokens_per_second": (
            sum(completion_tokens) / elapsed_seconds if completion_tokens and elapsed_seconds > 0 else None
        ),
    }


def prompt_for(kind: str, request_id: int) -> str:
    if kind == "short":
        return f"Explain KV cache in one sentence. Request id: {request_id}."
    if kind == "long":
        prefix = "A deterministic benchmark prompt about Prefill and Decode. " * 160
        return f"{prefix}\nQuestion {request_id}: summarize the distinction in one sentence."
    prefix = "System context: cache keys depend on exactly reproducible inputs.\n" * 80
    return f"{prefix}\nUser question {request_id}: explain one risk of dynamic fields."


def parse_sse_event(line: str) -> dict[str, object] | None:
    """Return a recognizable JSON SSE event, ignoring comments and malformed data."""
    if not line.startswith("data:"):
        return None
    data = line[5:].strip()
    if not data or data == "[DONE]":
        return None
    try:
        event = json.loads(data)
    except json.JSONDecodeError:
        return None
    if not isinstance(event, dict):
        return None
    if not isinstance(event.get("choices"), list) and not isinstance(event.get("usage"), dict):
        return None
    return event


def stream_request(args: argparse.Namespace, request_id: int) -> Record:
    body = {
        "model": args.model,
        "messages": [{"role": "user", "content": prompt_for(args.prompt_kind, request_id)}],
        "temperature": 0,
        "max_tokens": args.max_tokens,
        "stream": True,
        "stream_options": {"include_usage": True},
    }
    payload = json.dumps(body, separators=(",", ":")).encode("utf-8")
    request = urllib.request.Request(
        f"{args.base_url.rstrip('/')}/v1/chat/completions",
        data=payload,
        headers={"Content-Type": "application/json", "Accept": "text/event-stream"},
        method="POST",
    )
    started = time.perf_counter()
    first_event: float | None = None
    completion_tokens: int | None = None
    valid_events = 0
    try:
        with urllib.request.urlopen(request, timeout=args.timeout_seconds) as response:
            for raw_line in response:
                line = raw_line.decode("utf-8", errors="replace").strip()
                event = parse_sse_event(line)
                if event is None:
                    continue
                valid_events += 1
                if first_event is None:
                    first_event = time.perf_counter()
                usage = event.get("usage")
                if isinstance(usage, dict) and isinstance(usage.get("completion_tokens"), int):
                    completion_tokens = usage["completion_tokens"]
        ended = time.perf_counter()
    except (urllib.error.URLError, TimeoutError, OSError) as error:
        ended = time.perf_counter()
        return Record(request_id, None, (ended - started) * 1000, None, None, str(error))

    if valid_events == 0:
        return Record(
            request_id,
            None,
            (ended - started) * 1000,
            None,
            None,
            "HTTP response contained no recognizable JSON SSE events",
        )

    ttft_ms = (first_event - started) * 1000 if first_event is not None else None
    e2e_ms = (ended - started) * 1000
    approximate_tpot = None
    if ttft_ms is not None and completion_tokens is not None and completion_tokens > 1:
        approximate_tpot = max(0.0, (e2e_ms - ttft_ms) / (completion_tokens - 1))
    return Record(request_id, ttft_ms, e2e_ms, completion_tokens, approximate_tpot)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base-url", default="http://127.0.0.1:8000", help="OpenAI-compatible server base URL")
    parser.add_argument("--model", default="", help="Model name accepted by the target server")
    parser.add_argument("--requests", type=int, default=20, help="Number of measured requests")
    parser.add_argument("--concurrency", type=int, default=1, help="Client-side concurrent requests")
    parser.add_argument("--max-tokens", type=int, default=64, help="Requested generation limit")
    parser.add_argument("--timeout-seconds", type=float, default=120.0, help="Per-request HTTP timeout")
    parser.add_argument("--prompt-kind", choices=("short", "long", "repeated-prefix"), default="repeated-prefix")
    parser.add_argument("--output", default="", help="Optional path for raw JSONL records; otherwise stdout")
    parser.add_argument("--self-test", action="store_true", help="Validate local summary logic without HTTP")
    args = parser.parse_args()
    if not args.self_test:
        if not args.model:
            parser.error("--model is required unless --self-test is used")
        if args.requests <= 0 or args.concurrency <= 0 or args.max_tokens <= 0:
            parser.error("--requests, --concurrency, and --max-tokens must be positive")
    return args


def self_test() -> None:
    assert parse_sse_event("event: ping") is None
    assert parse_sse_event("data: [DONE]") is None
    assert parse_sse_event("data: not-json") is None
    assert parse_sse_event('data: {"message": "not a chat chunk"}') is None
    assert parse_sse_event('data: {"choices": []}') == {"choices": []}

    records = [
        Record(0, 10.0, 30.0, 3, 10.0),
        Record(1, 20.0, 50.0, 4, 10.0),
        Record(2, None, 5.0, None, None, "synthetic failure"),
    ]
    report = summary(records, elapsed_seconds=2.0)
    assert report["successful_requests"] == 2
    assert report["failed_requests"] == 1
    assert report["ttft_p50_ms"] == 10.0
    assert report["completion_tokens"] == 7
    assert report["output_tokens_per_second"] == 3.5
    print(json.dumps(report, sort_keys=True))
    print("SELF-CHECK: PASS")


def main() -> None:
    args = parse_args()
    if args.self_test:
        self_test()
        return

    started = time.perf_counter()
    with concurrent.futures.ThreadPoolExecutor(max_workers=args.concurrency) as executor:
        records = list(executor.map(lambda request_id: stream_request(args, request_id), range(args.requests)))
    elapsed = time.perf_counter() - started

    output = open(args.output, "w", encoding="utf-8") if args.output else sys.stdout
    try:
        for record in records:
            print(json.dumps(asdict(record), sort_keys=True), file=output)
        print(json.dumps({"summary": summary(records, elapsed)}, sort_keys=True), file=output)
    finally:
        if args.output:
            output.close()

    if any(record.error for record in records):
        print("WARNING: failed requests are included above; do not compare this run as a clean benchmark.", file=sys.stderr)
    if not any(record.completion_tokens is not None for record in records):
        print("WARNING: server did not return stream usage; TPOT and token throughput are null.", file=sys.stderr)


if __name__ == "__main__":
    main()
