package shape

import (
	"fmt"
	"sync"
	"testing"
)

func TestSchemaConcurrentReuse(t *testing.T) {
	t.Parallel()

	schema := testUserSchema().Strict()
	const workers = 32
	const parsesPerWorker = 100

	var wait sync.WaitGroup
	errorsFound := make(chan error, workers)
	for worker := 0; worker < workers; worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for i := 0; i < parsesPerWorker; i++ {
				user, err := schema.Parse(map[string]any{
					"name":  " Pong ",
					"email": "pong@example.com",
					"age":   30,
				})
				if err != nil {
					errorsFound <- err
					return
				}
				if user.Name != "Pong" || user.Age != 30 {
					errorsFound <- fmt.Errorf("unexpected user: %#v", user)
					return
				}
			}
		}()
	}
	wait.Wait()
	close(errorsFound)
	for err := range errorsFound {
		t.Fatal(err)
	}
}
