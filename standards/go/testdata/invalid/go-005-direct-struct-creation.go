package testdata

type Repo struct{}
type Service struct {
	repo *Repo
}

func BuildService() *Service {
	return &Service{repo: &Repo{}}
}
