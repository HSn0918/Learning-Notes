#!/usr/bin/env python3
"""A stdlib-only sampling lab for temperature, nucleus sampling, and seeds."""

from __future__ import annotations

import math
import random
from collections.abc import Sequence


TOKENS = ("cache", "prefill", "decode", "router", "fallback")
LOGITS = (2.4, 1.6, 0.8, 0.2, -0.7)


def softmax(logits: Sequence[float], temperature: float) -> list[float]:
    if temperature <= 0:
        raise ValueError("temperature must be > 0 for this educational sampler")
    scaled = [value / temperature for value in logits]
    maximum = max(scaled)
    weights = [math.exp(value - maximum) for value in scaled]
    total = sum(weights)
    return [weight / total for weight in weights]


def nucleus(probabilities: Sequence[float], top_p: float) -> list[float]:
    if not 0 < top_p <= 1:
        raise ValueError("top_p must be in (0, 1]")
    ordered = sorted(enumerate(probabilities), key=lambda item: item[1], reverse=True)
    kept: list[tuple[int, float]] = []
    cumulative = 0.0
    for index, probability in ordered:
        kept.append((index, probability))
        cumulative += probability
        if cumulative >= top_p:
            break

    normalized = [0.0] * len(probabilities)
    kept_total = sum(probability for _, probability in kept)
    for index, probability in kept:
        normalized[index] = probability / kept_total
    return normalized


def sample(probabilities: Sequence[float], rng: random.Random) -> int:
    threshold = rng.random()
    cumulative = 0.0
    for index, probability in enumerate(probabilities):
        cumulative += probability
        if threshold < cumulative:
            return index
    return len(probabilities) - 1  # Floating-point roundoff fallback.


def render(probabilities: Sequence[float]) -> str:
    return ", ".join(f"{token}={probability:.3f}" for token, probability in zip(TOKENS, probabilities))


def run_checks() -> None:
    cold = softmax(LOGITS, temperature=0.35)
    normal = softmax(LOGITS, temperature=1.0)
    restricted = nucleus(normal, top_p=0.70)
    assert max(cold) > max(normal), "lower temperature should sharpen this distribution"
    assert 0 < sum(probability > 0 for probability in restricted) < len(TOKENS)
    assert math.isclose(sum(restricted), 1.0), "nucleus distribution must be renormalized"

    rng = random.Random(20260813)
    second = [sample(restricted, rng) for _ in range(8)]
    first_rng = random.Random(20260813)
    first = [sample(restricted, first_rng) for _ in range(8)]
    assert first == second, "same seed must reproduce the same draw stream"


def main() -> None:
    cold = softmax(LOGITS, temperature=0.35)
    normal = softmax(LOGITS, temperature=1.0)
    restricted = nucleus(normal, top_p=0.70)
    rng = random.Random(20260813)
    draws = [TOKENS[sample(restricted, rng)] for _ in range(12)]

    print("temperature=0.35:", render(cold))
    print("temperature=1.00:", render(normal))
    print("temperature=1.00, top_p=0.70:", render(restricted))
    print("seed=20260813 draws:", ", ".join(draws))
    run_checks()
    print("SELF-CHECK: PASS")


if __name__ == "__main__":
    main()
