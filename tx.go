package apollo

import (
	"github.com/salvionied/apollo/serialization"
	"github.com/salvionied/apollo/serialization/Transaction"
)

type ApolloTransaction struct {
	Apollo *Apollo
	Tx     Transaction.Transaction
}

func (tx *ApolloTransaction) Submit() (serialization.TransactionId, error) {
	return tx.Apollo.backend.SubmitTx(tx.Tx)
}

func (tx *ApolloTransaction) Sign() *ApolloTransaction {
	signatures := tx.Apollo.wallet.SignTx(tx.Tx)
	tx.Tx.TransactionWitnessSet = signatures
	return tx
}
