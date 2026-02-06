#!/usr/bin/env bats

setup() {
  export PATH="$BATS_TEST_DIRNAME/bin:$PATH"
}

@test "ignore fzf fails fast when no pod specified" {
  run env KUBEEXEC_IGNORE_FZF=1 go run ./cmd/kubeexec --context ctx -n ns
  [ "$status" -ne 0 ]
  [[ "$output" == *"pod not specified and fzf is disabled"* ]]
}

@test "dry-run passes command args" {
  run env KUBEEXEC_IGNORE_FZF=1 go run ./cmd/kubeexec --context ctx -n ns --dry-run --non-interactive app-1 -- echo hello world
  [ "$status" -eq 0 ]
  [[ "$output" == *"-- echo hello world"* ]]
}
