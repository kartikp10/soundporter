package porter

import "log"

type Porter interface {
	Auth() error
	Export() error
	Import() error
}

type Port struct {
	Platform string
	Porter   Porter
}

func NewPort(platform string, porter Porter) *Port {
	return &Port{
		Platform: platform,
		Porter:   porter,
	}
}

func (p *Port) Auth() error {
	return p.Porter.Auth()
}

func (p *Port) Export() error {
	log.Printf("Exporting playlists to %s...", p.Platform)
	return p.Porter.Export()
}

func (p *Port) Import() error {
	log.Printf("Importing playlists to %s...", p.Platform)
	return p.Porter.Import()
}
