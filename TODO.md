# TODO: Making the Go Instrumentation Demo Better

This project needs some work to be a proper demo of instrumentation overhead. Right now it's too simple and uses environment variables everywhere, which isn't great Go style.

## 1. Fix the Architecture

**What's wrong**: Everything is controlled by `INSTRUMENTATION` env var, which is messy.

**What to do**:
- [ ] Create separate binaries in `cmd/` folders:
  - `cmd/baseline/` - no instrumentation
  - `cmd/manual/` - manual OpenTelemetry
  - `cmd/orchestrion/` - Datadog's compile-time tool  
  - `cmd/ebpf/` - runtime eBPF instrumentation
- [ ] Move shared code to `internal/` packages
- [ ] Use proper config files instead of env vars
- [ ] Update Makefile and docker-compose to use different binaries

## 2. Better Metrics

**What's wrong**: Only basic CPU/memory stats. Can't see real instrumentation impact.

**What to add**:
- [ ] Go runtime metrics (GC pauses, allocations, goroutines)
- [ ] Host system metrics (full resource usage)
- [ ] OpenTelemetry collector metrics (span rates, pipeline stats)
- [ ] Instrumentation overhead measurements

## 3. More Realistic Workloads

**What's wrong**: Just HTTP + simple DB calls. Too easy.

**What to add**:
- [ ] CPU-heavy work (JSON processing, crypto)
- [ ] Memory pressure (big allocations, GC stress)
- [ ] I/O operations (file system, network calls)
- [ ] Concurrent patterns (worker pools, channels)
- [ ] Error scenarios with proper context

## 4. Better Testing Setup

**What's wrong**: Everything runs locally. Hard to see real differences.

**What to do**:
- [ ] Add Lima VM config for consistent testing
- [ ] Simple Terraform for cloud VMs when needed
- [ ] Separate k6 from target apps
- [ ] Run tests in parallel to save time

## 5. Datadog Integration

**What's missing**: Only have Grafana dashboards.

**What to add**:
- [ ] Configure OTel collector to send to Datadog
- [ ] Create Datadog dashboard templates
- [ ] Compare metrics side-by-side with Grafana

## 6. Statistical Analysis

**What's wrong**: No proper statistical comparison.

**What to add**:
- [ ] Multiple test runs with averages
- [ ] Statistical significance testing
- [ ] Confidence intervals
- [ ] Performance regression detection

## Quick Wins (Start Here)

1. **Separate the binaries** - biggest bang for buck
2. **Add Go runtime metrics** - shows real overhead  
3. **Create Lima config** - consistent test environment
4. **Better workloads** - CPU and memory intensive stuff

## Lima Setup

Create `lima.yaml`:
```yaml
images:
- location: "https://cloud-images.ubuntu.com/releases/22.04/release/ubuntu-22.04-server-cloudimg-amd64.img"

memory: "4GiB"
cpus: 4

mounts:
- location: "."
  writable: true

containerd:
  system: false
  user: false

provision:
- mode: system
  script: |
    apt-get update
    apt-get install -y docker.io golang-1.21
```

## File Structure

```
cmd/
  baseline/main.go
  manual/main.go  
  orchestrion/main.go
  ebpf/main.go
internal/
  server/
  handlers/
  workloads/
  metrics/
configs/
  baseline.yaml
  manual.yaml
  orchestrion.yaml
  ebpf.yaml
terraform/
  aws/
  gcp/
lima.yaml
```

## Success Criteria

- Can easily compare instrumentation overhead
- Tests are reproducible across environments  
- Clear performance differences between approaches
- Proper statistical analysis of results
- Works both locally (Lima) and in cloud (Terraform)
- Datadog and Grafana show same insights

Keep it simple but effective.