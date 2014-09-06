package command

type Event interface {
	FileName() string
}

type event struct {
	Name string
}

func (e *event) FileName() string {
	return e.Name
}
