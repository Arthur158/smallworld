"""Fail early if an explicitly requested NVIDIA GPU cannot actually compute."""
import argparse
import json
import shutil
import subprocess
import sys

import torch


def nvidia_report():
    binary = shutil.which("nvidia-smi")
    if not binary:
        return "nvidia-smi not found"
    try:
        result = subprocess.run(
            [binary, "--query-gpu=name,driver_version,memory.total", "--format=csv,noheader"],
            capture_output=True, text=True, timeout=10, check=False,
        )
        return (result.stdout or result.stderr).strip()
    except (OSError, subprocess.TimeoutExpired) as exc:
        return str(exc)


def select_device(requested="auto"):
    if requested not in ("auto", "cpu", "cuda") and not requested.startswith("cuda:"):
        raise ValueError("device must be auto, cpu, cuda, or cuda:N")
    device = torch.device("cuda" if requested == "auto" and torch.cuda.is_available()
                          else "cpu" if requested == "auto" else requested)
    report = {"torch": str(torch.__version__), "torch_cuda": torch.version.cuda,
              "cuda_available": torch.cuda.is_available(), "nvidia_smi": nvidia_report(),
              "requested": requested, "device": str(device)}
    if device.type == "cuda":
        try:
            # Availability alone does not prove that this wheel supports the GPU.
            # Exercise allocation, matrix multiplication, backward, and synchronize.
            a = torch.randn(64, 64, device=device, requires_grad=True)
            loss = (a @ a.T).square().mean()
            loss.backward()
            torch.cuda.synchronize(device)
            if not torch.isfinite(loss) or not torch.isfinite(a.grad).all():
                raise RuntimeError("non-finite CUDA smoke-test result")
            props = torch.cuda.get_device_properties(device)
            report.update(gpu=props.name, memory_gib=round(props.total_memory / 2**30, 2),
                          capability=list(torch.cuda.get_device_capability(device)),
                          cuda_smoke_test="passed")
        except Exception as exc:
            raise RuntimeError(
                f"CUDA compute test failed: {exc}\n{json.dumps(report, indent=2)}\n"
                "Check the NVIDIA driver and install the wheel for your GPU from "
                "https://pytorch.org/get-started/locally/. Use --device cpu only if intended."
            ) from exc
    elif requested == "auto":
        report["notice"] = "CUDA is unavailable; using CPU. Use --device cuda to require the GPU."
    return device, report


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--device", default="auto")
    args = parser.parse_args()
    try:
        _, report = select_device(args.device)
        print(json.dumps(report, indent=2))
    except (RuntimeError, ValueError) as exc:
        print(str(exc), file=sys.stderr)
        raise SystemExit(1)


if __name__ == "__main__":
    main()
