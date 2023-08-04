package renderer

type GraphConfig struct {
	width   int
	height  int
	showSMA bool
}

func NewGraphConfig() *GraphConfig {
	return &GraphConfig{
		width:   512,
		height:  512,
		showSMA: true,
	}
}

func (g *GraphConfig) Width(width int) *GraphConfig {
	g.width = width
	return g
}

func (g *GraphConfig) Height(height int) *GraphConfig {
	g.height = height
	return g
}

func (g *GraphConfig) WithSMA(show bool) *GraphConfig {
	g.showSMA = show
	return g
}
