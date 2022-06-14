# AWS EKS E2E

## Debugging the test

For debugging environment provisioning, enable verbose output with `-verbose`
test flag.

```console
$ make test GO_TEST_ARGS="-verbose"
```

Th test environment is destroyed at the end by default. Run the tests with
`-retain` flag to retain the created test infrastructure.

```console
$ make test GO_TEST_ARGS="-retain"
```

The tests require the infrastructure state to be clean. For re-running the tests
with a retained infrastructure set `-existing` flag.

```console
$ make test GO_TEST_ARGS="-retain -existing"
```

To delete an existing infrastructure created with `-retain` flag:

```console
$ make test GO_TEST_ARGS="-existing"
```
