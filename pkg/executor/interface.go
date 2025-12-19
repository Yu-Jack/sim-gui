package executor

type Executor interface {
	Exec(command []string, env []string) (string, string, error)
}
