package porter

type Porter interface {
	Auth() error
	Export() error
	Import() error
}

type Port struct {
	Porter Porter
}

func NewPort(porter Porter) *Port {
	return &Port{
		Porter: porter,
	}
}

func (p *Port) Auth() error {
	return p.Porter.Auth()
}

func (p *Port) Export(destPath string) error {
	return p.Porter.Export()
}

func (p *Port) Import(sourcePath string) error {
	return p.Porter.Import()
}
