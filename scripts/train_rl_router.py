import json
import math
import random
from datetime import datetime, timezone


FEATURE_ORDER = [
    "link_quality",
    "latency_score",
    "bandwidth",
    "contact_active",
    "path_match",
    "energy_score",
]


def clamp(value, low=0.0, high=1.0):
    return max(low, min(high, value))


def sample_neighbor():
    link_quality = random.uniform(0.4, 1.0)
    latency_ms = random.uniform(10, 14000)
    bandwidth = random.uniform(50_000, 10_000_000)
    contact_active = random.choice([0.0, 1.0])
    path_match = random.choice([0.0, 1.0])
    energy_score = random.uniform(0.0, 1.0)

    latency_score = 1.0 - min(latency_ms / 10000.0, 1.0)
    bandwidth_score = min(bandwidth / 1_000_000.0, 1.0)
    return [
        link_quality,
        latency_score,
        bandwidth_score,
        contact_active,
        path_match,
        energy_score,
    ]


def reward(priority, features):
    link_quality, latency_score, bandwidth_score, contact_active, path_match, energy_score = features
    if energy_score < MIN_ENERGY_BY_PRIORITY[str(priority)]:
        return -1.0
    weights = PRIORITY_REWARD_WEIGHTS[str(priority)]
    value = sum(w * f for w, f in zip(weights, features))
    return value


def optimize_weights(iterations=5000, samples_per_iter=200):
    best = {}
    for priority in [0, 1, 2]:
        best_score = -math.inf
        best_weights = None
        for _ in range(iterations):
            weights = [random.random() for _ in FEATURE_ORDER]
            total = sum(weights)
            weights = [w / total for w in weights]
            score = 0.0
            for _ in range(samples_per_iter):
                features = sample_neighbor()
                score += reward(priority, features) * sum(w * f for w, f in zip(weights, features))
            score /= samples_per_iter
            if score > best_score:
                best_score = score
                best_weights = weights
        best[str(priority)] = best_weights
    return best


MIN_ENERGY_BY_PRIORITY = {"0": 0.3, "1": 0.2, "2": 0.1}
PRIORITY_REWARD_WEIGHTS = {
    "0": [0.2, 0.1, 0.4, 0.1, 0.1, 0.1],
    "1": [0.3, 0.25, 0.2, 0.1, 0.1, 0.05],
    "2": [0.25, 0.4, 0.1, 0.1, 0.1, 0.05],
}


def main():
    random.seed(42)
    optimized = optimize_weights()
    model = {
        "version": 1,
        "trained_at": datetime.now(timezone.utc).isoformat(),
        "feature_order": FEATURE_ORDER,
        "priority_weights": optimized,
        "min_energy_by_priority": MIN_ENERGY_BY_PRIORITY,
        "notes": "Generated via lightweight RL-inspired optimizer.",
    }

    with open("models/rl_router.json", "w", encoding="utf-8") as handle:
        json.dump(model, handle, indent=2)

    print("RL router model written to models/rl_router.json")


if __name__ == "__main__":
    main()
