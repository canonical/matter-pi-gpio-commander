# Run Tests

```bash
go test -v -failfast -count 1
```

where:
- `-v` is to enable verbose output
- `-failfast` makes the test stop after first failure
- `-count 1` is to avoid Go test caching for example when testing a rebuilt snap
