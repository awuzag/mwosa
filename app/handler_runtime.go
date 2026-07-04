package app

import "github.com/awuzag/mwosa/app/handler"

func newHandlers(services ServiceRuntime) (Handlers, error) {
	return Handlers{
		Aggregates:   handler.NewAggregate(services.Aggregates),
		Backtest:     handler.NewBacktest(services.Backtest),
		Compositions: handler.NewComposition(services.Compositions),
		Daily:        handler.NewDaily(services.Daily.Reader, services.Daily.Collector),
		Index:        handler.NewIndex(services.Index.Reader, services.Index.Collector),
		Macro:        handler.NewMacro(services.Macro.Reader, services.Macro.Collector),
		Financials:   handler.NewFinancials(services.Financials),
		Instruments:  handler.NewInstrument(services.Instruments),
		Intraday:     handler.NewIntraday(services.Intraday),
		Migration:    handler.NewMigration(services.Migration),
		Orderbooks:   handler.NewOrderbook(services.Orderbooks),
		Quotes:       handler.NewQuote(services.Quotes),
		Strategy:     handler.NewStrategy(services.Strategy, services.Universe),
		Trades:       handler.NewTrades(services.Trades),
	}, nil
}
