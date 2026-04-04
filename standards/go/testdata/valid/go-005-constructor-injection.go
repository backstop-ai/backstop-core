package testdata

type Store interface {
	Ping() error
}

type App struct {
	component Store
}

func NewApp(store Store) *App {
	return &App{component: store}
}
