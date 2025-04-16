package model

type PositionList struct {
	Category       string     `json:"category"`
	List           []Position `json:"list"`
	NextPageCursor string     `json:"nextPageCursor"`
}

type Position struct {
	AdlRankIndicator float64 `json:"adlRankIndicator"`
	AutoAddMargin    float64 `json:"autoAddMargin"`
	AvgPrice         string  `json:"avgPrice"`
	BustPrice        string  `json:"bustPrice"`
	CreatedTime      string  `json:"createdTime"`
	CumRealisedPnl   string  `json:"cumRealisedPnl"`
	CurRealisedPnl   string  `json:"curRealisedPnl"`
	IsReduceOnly     bool    `json:"isReduceOnly"`
	Leverage         string  `json:"leverage"`
	LiqPrice         string  `json:"liqPrice"`
	MarkPrice        string  `json:"markPrice"`
	PositionBalance  string  `json:"positionBalance"`
	PositionIM       string  `json:"positionIM"`
	PositionIdx      int     `json:"positionIdx"`
	PositionMM       string  `json:"positionMM"`
	PositionStatus   string  `json:"positionStatus"`
	PositionValue    string  `json:"positionValue"`
	RiskId           float64 `json:"riskId"`
	RiskLimitValue   string  `json:"riskLimitValue"`
	Seq              float64 `json:"seq"`
	SessionAvgPrice  string  `json:"sessionAvgPrice"`
	Side             string  `json:"side"`
	Size             string  `json:"size"`
	StopLoss         string  `json:"stopLoss"`
	Symbol           string  `json:"symbol"`
	TakeProfit       string  `json:"takeProfit"`
	TpslMode         string  `json:"tpslMode"`
	TradeMode        float64 `json:"tradeMode"`
	TrailingStop     string  `json:"trailingStop"`
	UnrealisedPnl    string  `json:"unrealisedPnl"`
	UpdatedTime      string  `json:"updatedTime"`
}
