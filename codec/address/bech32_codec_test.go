package address

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"testing"
)

func TestName(t *testing.T) {
	address := "enivaloper1cq0fwyyxr7ynp7gyqmzpje354jfm5lwh6cp45j"
	add, err := NewBech32Codec("enivaloper").StringToBytes(address)
	if err != nil {
		t.Fatal(err)
	}
	println(add)
	println(sdk.AccAddress(add).String())
}
