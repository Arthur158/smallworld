"""Export an actor to data-only JSON for the Go room worker."""
import argparse
from .model import export_policy, load_checkpoint


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("checkpoint")
    parser.add_argument("--output", required=True)
    parser.add_argument("--id", required=True, help="immutable versioned model id, e.g. ppo-v1")
    parser.add_argument("--name", required=True)
    parser.add_argument("--description", default="Trained PPO policy")
    args = parser.parse_args()
    policy, _ = load_checkpoint(args.checkpoint)
    export_policy(policy, args.output, args.id, args.name, args.description)
    print(args.output)


if __name__ == "__main__":
    main()
