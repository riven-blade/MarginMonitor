package exchange

type Exchange interface {
	FetchPositions() (interface{}, error)
	AddMargin(symbol string, amount float64) string
	GetName() string
}
