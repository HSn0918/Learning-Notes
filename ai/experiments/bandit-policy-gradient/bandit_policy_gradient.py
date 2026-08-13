#!/usr/bin/env python3
"""A tiny REINFORCE example with a two-armed Bernoulli bandit."""

from __future__ import annotations

import math
import random


REWARD_PROBABILITIES = (0.20, 0.80)


def softmax(logits: list[float]) -> list[float]:
    maximum = max(logits)
    weights = [math.exp(value - maximum) for value in logits]
    total = sum(weights)
    return [weight / total for weight in weights]


def choose(probabilities: list[float], rng: random.Random) -> int:
    return 0 if rng.random() < probabilities[0] else 1


def train(*, seed: int = 20260813, episodes: int = 4000, learning_rate: float = 0.08) -> tuple[list[float], float]:
    rng = random.Random(seed)
    logits = [0.0, 0.0]
    baseline = 0.0
    recent_rewards: list[float] = []

    for _ in range(episodes):
        probabilities = softmax(logits)
        action = choose(probabilities, rng)
        reward = 1.0 if rng.random() < REWARD_PROBABILITIES[action] else 0.0
        advantage = reward - baseline
        # d log pi(action) / d logits[i] = 1[action == i] - pi[i]
        for index, probability in enumerate(probabilities):
            gradient = (1.0 if index == action else 0.0) - probability
            logits[index] += learning_rate * advantage * gradient
        baseline = 0.99 * baseline + 0.01 * reward
        recent_rewards.append(reward)

    return softmax(logits), sum(recent_rewards[-500:]) / 500


def main() -> None:
    before = softmax([0.0, 0.0])
    after, recent_reward = train()
    print(f"reward probabilities: arm-0={REWARD_PROBABILITIES[0]:.2f}, arm-1={REWARD_PROBABILITIES[1]:.2f}")
    print(f"policy before: arm-0={before[0]:.3f}, arm-1={before[1]:.3f}")
    print(f"policy after:  arm-0={after[0]:.3f}, arm-1={after[1]:.3f}")
    print(f"last 500 episodes average reward: {recent_reward:.3f}")
    assert after[1] > 0.80, "policy should prefer the higher-reward arm for the fixed seed"
    assert after[1] > after[0]
    assert recent_reward > 0.65
    print("SELF-CHECK: PASS")


if __name__ == "__main__":
    main()
