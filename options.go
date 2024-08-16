package mongodebug

type Option func(*debugSession)

func WithThreshold(n int) Option {
	return func(ds *debugSession) {
		ds.threshold = n
	}
}
