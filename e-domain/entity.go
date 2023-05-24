package e_domain

type Server interface {
	ID() string
	Name() string
	Flavor() Flavor
	Volumes() []Volume
	Networks() []Network
}

type Flavor interface {
	ID() string
	Name() string
}

type Volume interface {
	ID() string
	Name() string
}

type Network interface {
	ID() string
	Name() string
}

func ServerInstance(id string, name string, flavor Flavor) Server {
	return &serverEntity{
		id:     id,
		name:   name,
		flavor: flavor,
	}
}

type serverEntity struct {
	id       string
	name     string
	flavor   Flavor
	volumes  []Volume
	networks []Network
}

func (s *serverEntity) ID() string {
	return s.id
}

func (s *serverEntity) Name() string {
	return s.name
}

func (s *serverEntity) Flavor() Flavor {
	return s.flavor
}

func (s *serverEntity) Volumes() []Volume {
	//TODO implement me
	panic("implement me")
}

func (s *serverEntity) Networks() []Network {
	//TODO implement me
	panic("implement me")
}

type flavorEntity struct {
	id   string
	name string
}

func FlavorInstance(id string, name string) *flavorEntity {
	return &flavorEntity{id: id, name: name}
}

func (f *flavorEntity) ID() string {
	return f.id
}

func (f *flavorEntity) Name() string {
	return f.name
}
