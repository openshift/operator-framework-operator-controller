#!/usr/bin/env python3

"""Watch active and cumulative cardinality of the OLM metrics spike."""

import argparse
import datetime
import subprocess
import sys
import time

METRICS = (
    "olm_clusterextension_info",
    "olm_clusterextension_condition",
    "olm_cluster_catalog_serving",
    "olm_cluster_catalog_condition",
)
BLOCKS = "▁▂▃▄▅▆▇█"


def parse_labels(value):
    labels = {}
    i = 0
    while i < len(value):
        equals = value.find("=", i)
        if equals < 0 or equals + 1 >= len(value) or value[equals + 1] != '"':
            raise ValueError(f"invalid label set near {value[i:]!r}")
        name = value[i:equals]
        i = equals + 2
        chars = []
        while i < len(value) and value[i] != '"':
            if value[i] == "\\":
                i += 1
                if i >= len(value):
                    raise ValueError("unterminated label escape")
                chars.append("\n" if value[i] == "n" else value[i])
            else:
                chars.append(value[i])
            i += 1
        if i >= len(value):
            raise ValueError("unterminated label value")
        labels[name] = "".join(chars)
        i += 1
        if i < len(value):
            if value[i] != ",":
                raise ValueError(f"invalid label separator {value[i]!r}")
            i += 1
    return labels


def samples(text, metric_names=METRICS):
    result = {name: [] for name in metric_names}
    for line in text.splitlines():
        name, separator, remainder = line.partition("{")
        if not separator or name not in result:
            continue
        label_end = remainder.rfind("}")
        if label_end < 0:
            raise ValueError(f"missing closing brace in {line!r}")
        try:
            value = float(remainder[label_end + 1 :].strip().split()[0])
        except (IndexError, ValueError) as error:
            raise ValueError(f"invalid sample value in {line!r}") from error
        result[name].append({"labels": parse_labels(remainder[:label_end]), "value": value})
    return result


def sparkline(values, width=60):
    values = values[-width:]
    if not values:
        return ""
    low, high = min(values), max(values)
    if low == high:
        return BLOCKS[0] * len(values)
    return "".join(BLOCKS[round((value - low) * (len(BLOCKS) - 1) / (high - low))] for value in values)


def create_token(namespace, service_account):
    try:
        return subprocess.run(
            ["kubectl", "create", "token", service_account, "-n", namespace],
            check=True,
            capture_output=True,
            text=True,
        ).stdout.strip()
    except subprocess.CalledProcessError as error:
        raise SystemExit(f"kubectl could not create a token for {service_account}: {error.stderr.strip()}") from error


def start_forward(namespace, service, local_port, remote_port):
    return subprocess.Popen(
        ["kubectl", "port-forward", "-n", namespace, f"svc/{service}", f"{local_port}:{remote_port}"],
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
    )


def fetch(port, token):
    return subprocess.run(
        [
            "curl",
            "--fail",
            "--silent",
            "--show-error",
            "--insecure",
            "--config",
            "-",
            f"https://127.0.0.1:{port}/metrics",
        ],
        check=True,
        input=f'header = "Authorization: Bearer {token}"\n',
        capture_output=True,
        text=True,
        timeout=3,
    ).stdout


def update_state(state, current):
    fingerprints = {tuple(sorted(sample["labels"].items())) for sample in current}
    new = fingerprints - state["seen"]
    state["seen"].update(fingerprints)
    for sample in current:
        for name, value in sample["labels"].items():
            state["seen_values"].setdefault(name, set()).add(value)
    state["active_history"].append(len(fingerprints))
    state["seen_history"].append(len(state["seen"]))
    return fingerprints, new


def print_metric(name, current, state):
    fingerprints, new = update_state(state, current)
    values = sorted({sample["value"] for sample in current})
    print(f"\n{name}")
    print(f"active: {len(fingerprints):<5} unique seen: {len(state['seen']):<5} new: +{len(new):<4} values: {values}")
    print(f"active  {sparkline(state['active_history'])}")
    print(f"seen    {sparkline(state['seen_history'])}")
    if not current:
        print("No series found. Rebuild/redeploy with `make run` and verify the resource exists.")
        return

    print("label                            active distinct   seen distinct")
    print("-" * 65)
    label_names = sorted({label for sample in current for label in sample["labels"]})
    for label in label_names:
        active_values = {sample["labels"].get(label, "") for sample in current}
        print(f"{label:<32} {len(active_values):>8} {len(state['seen_values'][label]):>15}")


def self_test():
    parsed = samples(
        '# HELP ignored\n'
        'olm_clusterextension_info{name="example",channels="preview,stable"} 1\n'
        'olm_cluster_catalog_serving{name="catalog",digest="",reason="Unavailable"} 0\n'
    )
    assert parsed["olm_clusterextension_info"] == [
        {"labels": {"name": "example", "channels": "preview,stable"}, "value": 1.0}
    ]
    assert parsed["olm_cluster_catalog_serving"][0]["value"] == 0
    assert sparkline([1, 2, 3]) == "▁▅█"
    print("self-test passed")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--interval", type=float, default=2, help="seconds between scrapes (default: 2)")
    parser.add_argument("--port", type=int, default=18443, help="operator-controller local port (default: 18443)")
    parser.add_argument("--catalog-port", type=int, default=17443, help="catalogd local port (default: 17443)")
    parser.add_argument("--namespace", default="olmv1-system")
    parser.add_argument("--once", action="store_true", help="scrape once without clearing the terminal")
    parser.add_argument("--self-test", action="store_true")
    args = parser.parse_args()

    if args.self_test:
        self_test()
        return

    token = create_token(args.namespace, "operator-controller-controller-manager")
    endpoints = (
        {
            "name": "operator-controller",
            "port": args.port,
            "remote_port": 8443,
            "service": "operator-controller-service",
            "token": token,
            "metrics": METRICS[:2],
        },
        {
            "name": "catalogd",
            "port": args.catalog_port,
            "remote_port": 7443,
            "service": "catalogd-service",
            "token": token,
            "metrics": METRICS[2:],
        },
    )
    forwards = [
        start_forward(args.namespace, endpoint["service"], endpoint["port"], endpoint["remote_port"])
        for endpoint in endpoints
    ]

    # ponytail: cumulative history is intentionally in-memory; use Prometheus when persistence is needed.
    states = {
        metric: {"seen": set(), "seen_values": {}, "active_history": [], "seen_history": []}
        for metric in METRICS
    }
    scrape_count = 0

    try:
        while True:
            try:
                current = {}
                for endpoint in endpoints:
                    current.update(samples(fetch(endpoint["port"], endpoint["token"]), endpoint["metrics"]))
            except Exception as error:
                stopped = [
                    (endpoint, forward)
                    for endpoint, forward in zip(endpoints, forwards, strict=True)
                    if forward.poll() is not None
                ]
                if stopped:
                    endpoint, forward = stopped[0]
                    output = forward.stdout.read() if forward.stdout else ""
                    raise RuntimeError(f"{endpoint['name']} port-forward exited:\n{output}") from error
                print(f"waiting for metrics endpoints: {error}", file=sys.stderr)
                time.sleep(0.5)
                continue

            scrape_count += 1
            if not args.once and sys.stdout.isatty():
                print("\033[2J\033[H", end="")
            print(f"OLM metrics cardinality — {datetime.datetime.now().astimezone().strftime('%H:%M:%S %Z')} — scrape {scrape_count}")
            for metric in METRICS:
                print_metric(metric, current[metric], states[metric])

            if args.once:
                return
            time.sleep(args.interval)
    except KeyboardInterrupt:
        pass
    finally:
        for forward in forwards:
            forward.terminate()
        for forward in forwards:
            try:
                forward.wait(timeout=3)
            except subprocess.TimeoutExpired:
                forward.kill()


if __name__ == "__main__":
    main()
