#!/usr/bin/env python3
"""Parameter-count intuition for a dense projection and its LoRA update."""

from __future__ import annotations


def dense_parameters(d_in: int, d_out: int) -> int:
    return d_in * d_out


def lora_parameters(d_in: int, d_out: int, rank: int) -> int:
    if not 0 < rank <= min(d_in, d_out):
        raise ValueError("rank must be in [1, min(d_in, d_out)]")
    return rank * d_in + d_out * rank


def format_count(count: int) -> str:
    return f"{count:,}"


def main() -> None:
    d_in = d_out = 1024
    dense = dense_parameters(d_in, d_out)
    print(f"one {d_out}x{d_in} projection: dense trainable={format_count(dense)}")
    for rank in (4, 16, 64, 256):
        adapter = lora_parameters(d_in, d_out, rank)
        print(
            f"rank={rank:>3}: LoRA trainable={format_count(adapter):>9} "
            f"({adapter / dense:.2%} of dense)"
        )

    layers, projections, rank = 24, 7, 16
    full_stack = layers * projections * dense
    lora_stack = layers * projections * lora_parameters(d_in, d_out, rank)
    print(
        f"simplified {layers}-layer x {projections}-projection stack: "
        f"dense={format_count(full_stack)}, LoRA={format_count(lora_stack)}"
    )

    assert lora_parameters(d_in, d_out, 16) == 2 * 1024 * 16
    assert lora_parameters(d_in, d_out, 16) < dense
    assert lora_parameters(d_in, d_out, 64) > lora_parameters(d_in, d_out, 16)
    assert lora_parameters(d_in, d_out, 1024) == 2 * dense
    print("SELF-CHECK: PASS")


if __name__ == "__main__":
    main()
