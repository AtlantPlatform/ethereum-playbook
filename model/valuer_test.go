package model

import (
	"context"
	"math/big"
	"testing"

	"github.com/AtlantPlatform/ethfw"
	"github.com/stretchr/testify/assert"
)

func TestValuerParse(t *testing.T) {
	assert := assert.New(t)

	ctx := AppContext{context.Background()}
	ev, err := Valuer("5").Parse(ctx, nil, nil)
	if assert.NoError(err) {
		assert.EqualValues(ev.Value, big.NewInt(5))
		assert.EqualValues(ev.ValueWei, ethfw.BigWei(big.NewInt(5)))
		assert.EqualValues(ev.Denominator, "")
	}
	ev, err = Valuer("0.5 ether").Parse(ctx, nil, nil)
	assert.Error(err)

	ev, err = Valuer("5 * 1e8 * 1e9").Parse(ctx, nil, nil)
	if assert.NoError(err) {
		expected := ethfw.ToWei(0.5).ToInt()
		assert.EqualValues(expected, ev.Value)
		assert.EqualValues(expected, ev.ValueWei.ToInt())
		assert.EqualValues(ev.Denominator, "")
	}

	ev, err = Valuer("5 * 1e8 gwei").Parse(ctx, nil, nil)
	if assert.NoError(err) {
		expected := ethfw.ToWei(0.5).ToInt()
		assert.EqualValues(expected, ev.Value)
		assert.EqualValues(expected, ev.ValueWei.ToInt())
		assert.EqualValues(ev.Denominator, "gwei")
	}
}

func TestDenominateValue(t *testing.T) {
	assert := assert.New(t)

	input := []struct {
		Input       *big.Int
		Denominator string
		Expected    *big.Int
	}{
		{big.NewInt(1), "wei", big.NewInt(1)},
		{big.NewInt(1), "gwei", ethfw.Gwei(1).ToInt()},
		{big.NewInt(1), "ether", ethfw.ToWei(1).ToInt()},
		{big.NewInt(1), "eth", big.NewInt(1)},
		{big.NewInt(1), "kek", big.NewInt(1)},
		{big.NewInt(int64(5e8)), "gwei", ethfw.ToWei(0.5).ToInt()},
	}
	for _, in := range input {
		assert.EqualValues(in.Expected, denominateValue(in.Input, in.Denominator))
	}
}

func TestIsCommonDenominator(t *testing.T) {
	assert := assert.New(t)

	assert.True(IsCommonDenominator("wei"))
	assert.True(IsCommonDenominator("gwei"))
	assert.True(IsCommonDenominator("ether"))
	assert.True(IsCommonDenominator("eth"))
	assert.False(IsCommonDenominator("kek"))
}
