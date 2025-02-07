package greeks

import (
	"math"
)

type Greeks struct {
	Gamma float64
	Delta float64
	Vega  float64
	Theta float64
	Rho   float64
	Iv    float64
}

type Option struct {
	TradingSymbol string
	Strike        float64
	Type          string
	MidPrice      float64
}

type OptionChain struct {
	Name              string
	Spot              float64
	Future            float64
	Vix               float64
	Exchange          string
	HoursToExpiry     float64
	LastTradedTime    int64
	LastTimestamp     int64
	ReceivedTimestamp int64
	SinkTimestamp     int64
	StrikeDifference  float64
	IsGreeksUpdated   bool
	Options           map[string]*Option
}

func RunGreek(
	underlyingPrice, hoursLeft float64,
	strike float64,
	price float64,
	callType bool,
) *Greeks {
	freeRate := 0.02
	if hoursLeft <= 0 {
		hoursLeft = 0.001
	}
	yearsLeft := hoursLeft / (250 * 6.25)

	iv := BSImpliedVol(callType, price, underlyingPrice, strike, yearsLeft, 0.0, freeRate, 0.0)

	return &Greeks{
		Delta: BSDelta(callType, underlyingPrice, strike, yearsLeft, iv, freeRate, 0.0),
		Vega:  BSVega(underlyingPrice, strike, yearsLeft, iv, freeRate, 0.0),
		Gamma: BSGamma(underlyingPrice, strike, yearsLeft, iv, freeRate, 0.0),
		Theta: BSTheta(callType, underlyingPrice, strike, yearsLeft, iv, freeRate, 0.0),
		Rho:   BSRho(callType, underlyingPrice, strike, yearsLeft, iv, freeRate, 0.0),
		Iv:    iv,
	}
}

func GetVix(oc *OptionChain) (float64, float64) {
	T := oc.HoursToExpiry / (250 * 6.25)
	F := oc.Future
	k := oc.StrikeDifference
	K0 := math.Floor(F/k) * k
	R := 0.02
	terms := 0.0
	termsItm := 0.0
	for _, o := range oc.Options {
		if (F > o.Strike && o.Type == "PE") || (F < o.Strike && o.Type == "CE") {
			terms += (k / math.Pow(o.Strike, 2)) * math.Exp(R*T) * o.MidPrice
			termsItm += (k / math.Pow(o.Strike, 2)) * math.Exp(R*T) * o.MidPrice
		} else if F < o.Strike && o.Type == "PE" {
			intrinsicValue := o.Strike - F
			extrinsicValue := o.MidPrice - intrinsicValue
			termsItm += (k / math.Pow(o.Strike, 2)) * math.Exp(R*T) * extrinsicValue
		} else if F > o.Strike && o.Type == "CE" {
			intrinsicValue := F - o.Strike
			extrinsicValue := o.MidPrice - intrinsicValue
			termsItm += (k / math.Pow(o.Strike, 2)) * math.Exp(R*T) * extrinsicValue
		}
	}
	variance := (2/T)*terms - (1/T)*math.Pow((F/K0-1), 2)
	vix := math.Sqrt(variance)
	varianceItm := (1/T)*termsItm - (1/T)*math.Pow((F/K0-1), 2)
	vixItm := math.Sqrt(varianceItm)
	return vix, vixItm
}

func GetAvar(oc *OptionChain) float64 {
	F := oc.Future
	terms := 0.0
	for _, o := range oc.Options {
		if o.Strike < F && o.Type == "PE" {
			terms -= o.Strike * o.MidPrice
		}

		if o.Strike >= F && o.Type == "CE" {
			terms += o.Strike * o.MidPrice
		}
	}
	return 2 * terms / math.Pow(F, 2)
}
