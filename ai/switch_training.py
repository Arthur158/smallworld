#!/usr/bin/env python3
"""Switch the GPU service between preserved runs and specialist training."""
import argparse
import shlex
import subprocess


def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('run',choices=['old','new','entity','specialists','specialists-previous','status'], help='entity/new: previous Moneta run; specialists: revised expert learning; specialists-previous: earlier modular run; old: earlier league run')
    p.add_argument('--ssh-command',default='ssh gpu',help='complete SSH connection command, e.g. ssh gpu')
    args=p.parse_args()
    command=shlex.split(args.ssh_command)
    if not command:p.error('--ssh-command cannot be empty')
    remote='bash "$HOME/github/smallworld/ai/switch-training-run.sh" '+shlex.quote(args.run)
    subprocess.run(command+[remote],check=True)


if __name__=='__main__':main()
