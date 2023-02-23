package email

type PaymentNotifyVar struct {
	Title     string
	Sender    string
	Receiver  string
	IsRequest bool
	Link      string
	Path      string
}
