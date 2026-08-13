#!/usr/bin/env python3
"""Show JSON byte determinism without claiming tokenizer/cache behaviour."""

from __future__ import annotations

import hashlib
import json


def encode(value: dict[str, object], *, canonical: bool) -> bytes:
    return json.dumps(
        value,
        ensure_ascii=False,
        separators=(",", ":"),
        sort_keys=canonical,
    ).encode("utf-8")


def digest(payload: bytes) -> str:
    return hashlib.sha256(payload).hexdigest()


def main() -> None:
    first = {"user_id": "u-123", "level": "VIP", "region": "cn"}
    second = {"region": "cn", "level": "VIP", "user_id": "u-123"}

    raw_first = encode(first, canonical=False)
    raw_second = encode(second, canonical=False)
    canonical_first = encode(first, canonical=True)
    canonical_second = encode(second, canonical=True)

    print("raw A:", raw_first.decode())
    print("raw B:", raw_second.decode())
    print("raw hashes:", digest(raw_first), digest(raw_second))
    print("canonical:", canonical_first.decode())
    print("canonical hash:", digest(canonical_first))

    assert raw_first != raw_second, "insertion order should be visible without canonicalization"
    assert digest(raw_first) != digest(raw_second)
    assert canonical_first == canonical_second
    assert digest(canonical_first) == digest(canonical_second)
    print("SELF-CHECK: PASS")


if __name__ == "__main__":
    main()
