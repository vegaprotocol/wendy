package list

type Option int

const (
	Backward Option = 1 << iota
)

type Options []Option

func (opts Options) all() Option {
	var all int
	for _, opt := range opts {
		all += int(opt)
	}
	return Option(all)
}
func (opts Options) Backward() bool { return (opts.all() & Backward) != 0 }
