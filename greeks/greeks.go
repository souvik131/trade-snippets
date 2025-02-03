package greeks

import (
	"fmt"
	"math"
)

var sqtwopi float64 = math.Sqrt(2 * math.Pi)
var IVPrecision = 0.00001

func PriceBlackScholes(callType bool, underlying float64, strike float64, timeToExpiration float64, volatility float64, riskFreeInterest float64, dividend float64) float64 {

	var sign float64
	if callType {
		if timeToExpiration <= 0 {
			return math.Abs(underlying - strike)
		}
		sign = 1
	} else {
		if timeToExpiration <= 0 {
			return math.Abs(strike - underlying)
		}
		sign = -1
	}

	if sign == 0 {
		return 0.0
	}

	re := math.Exp(-riskFreeInterest * timeToExpiration)
	qe := math.Exp(-dividend * timeToExpiration)
	vt := volatility * (math.Sqrt(timeToExpiration))
	d1 := d1f(underlying, strike, timeToExpiration, volatility, riskFreeInterest, dividend, vt)
	d2 := d2f(d1, vt)
	d1 = sign * d1
	d2 = sign * d2
	nd1 := Stdnorm.Cdf(d1)
	nd2 := Stdnorm.Cdf(d2)

	bsprice := sign * ((underlying * qe * nd1) - (strike * re * nd2))
	return bsprice
}

func d1f(underlying float64, strike float64, timeToExpiration float64, volatility float64, riskFreeInterest float64, dividend float64, volatilityWithExpiration float64) float64 {
	d1 := math.Log(underlying/strike) + (timeToExpiration * (riskFreeInterest - dividend + ((volatility * volatility) * 0.5)))
	d1 = d1 / volatilityWithExpiration
	return d1
}

func d2f(d1 float64, volatilityWithExpiration float64) float64 {
	d2 := d1 - volatilityWithExpiration
	return d2
}
func d1pdff(underlying float64, strike float64, timeToExpiration float64, volatility float64, riskFreeInterest float64, dividend float64) float64 {
	vt := volatility * (math.Sqrt(timeToExpiration))
	d1 := d1f(underlying, strike, timeToExpiration, volatility, riskFreeInterest, dividend, vt)
	d1pdf := math.Exp(-(d1 * d1) * 0.5)
	d1pdf = d1pdf / sqtwopi
	return d1pdf
}

func BSDelta(callType bool, underlying float64, strike float64, timeToExpiration float64, volatility float64, riskFreeInterest float64, dividend float64) float64 {
	var zo float64
	if !callType {
		zo = -1
	} else {
		zo = 0
	}

	drq := math.Exp(-dividend * timeToExpiration)
	vt := volatility * (math.Sqrt(timeToExpiration))
	d1 := d1f(underlying, strike, timeToExpiration, volatility, riskFreeInterest, dividend, vt)
	cdfd1 := Stdnorm.Cdf(d1)
	delta := drq * (cdfd1 + zo)
	return delta
}

func BSVega(underlying float64, strike float64, timeToExpiration float64, volatility float64, riskFreeInterest float64, dividend float64) float64 {
	d1pdf := d1pdff(underlying, strike, timeToExpiration, volatility, riskFreeInterest, dividend)
	drq := math.Exp(-dividend * timeToExpiration)
	sqt := math.Sqrt(timeToExpiration)
	vega := (d1pdf) * drq * underlying * sqt * 0.01
	return vega
}

func BSGamma(underlying float64, strike float64, timeToExpiration float64, volatility float64, riskFreeInterest float64, dividend float64) float64 {
	drq := math.Exp(-dividend * timeToExpiration)
	drd := underlying * volatility * math.Sqrt(timeToExpiration)
	d1pdf := d1pdff(underlying, strike, timeToExpiration, volatility, riskFreeInterest, dividend)
	gamma := (drq / drd) * d1pdf
	return gamma
}

func BSTheta(callType bool, underlying float64, strike float64, timeToExpiration float64, volatility float64, riskFreeInterest float64, dividend float64) float64 {

	var sign float64
	if !callType {
		sign = -1
	} else {
		sign = 1
	}

	sqt := math.Sqrt(timeToExpiration)
	drq := math.Exp(-dividend * timeToExpiration)
	dr := math.Exp(-riskFreeInterest * timeToExpiration)
	d1pdf := d1pdff(underlying, strike, timeToExpiration, volatility, riskFreeInterest, dividend)
	twosqt := 2 * sqt
	p1 := -1 * ((underlying * volatility * drq) / twosqt) * d1pdf

	vt := volatility * (sqt)
	d1 := d1f(underlying, strike, timeToExpiration, volatility, riskFreeInterest, dividend, vt)
	d2 := d2f(d1, vt)
	var nd1, nd2 float64

	d1 = sign * d1
	d2 = sign * d2
	nd1 = Stdnorm.Cdf(d1)
	nd2 = Stdnorm.Cdf(d2)

	p2 := -sign * riskFreeInterest * strike * dr * nd2
	p3 := sign * dividend * underlying * drq * nd1
	theta := (p1 + p2 + p3) / 365
	return theta
}

func BSRho(callType bool, underlying float64, strike float64, timeToExpiration float64, volatility float64, riskFreeInterest float64, dividend float64) float64 {
	var sign float64
	if !callType {
		sign = -1
	} else {
		sign = 1
	}

	dr := math.Exp(-riskFreeInterest * timeToExpiration)
	p1 := sign * (strike * timeToExpiration * dr) / 100

	vt := volatility * (math.Sqrt(timeToExpiration))
	d1 := d1f(underlying, strike, timeToExpiration, volatility, riskFreeInterest, dividend, vt)
	d2 := sign * d2f(d1, vt)
	nd2 := Stdnorm.Cdf(d2)
	rho := p1 * nd2
	return rho
}

func BSImpliedVol(callType bool, lastTradedPrice float64, underlying float64, strike float64, timeToExpiration float64, startAnchorVolatility float64, riskFreeInterest float64, dividend float64) float64 {
	if startAnchorVolatility > 0 == false {
		startAnchorVolatility = 0.5
	}
	errlim := IVPrecision
	maxl := 100
	dv := errlim + 1
	n := 0
	maxloops := 100

	for ; math.Abs(dv) > errlim && n < maxl; n++ {
		difval := PriceBlackScholes(callType, underlying, strike, timeToExpiration, startAnchorVolatility, riskFreeInterest, dividend) - lastTradedPrice
		v1 := BSVega(underlying, strike, timeToExpiration, startAnchorVolatility, riskFreeInterest, dividend) / 0.01
		dv = difval / v1
		startAnchorVolatility = startAnchorVolatility - dv
	}
	var iv float64
	if n < maxloops {
		iv = startAnchorVolatility
	} else {
		iv = math.NaN()
	}

	return iv
}

var sqrt2 float64 = math.Pow(2, 0.5)
var toomanydev float64 = 8

type normdist struct {
	stddev      float64
	mean        float64
	stddevsqpi  float64
	twostddevsq float64
}

func NewNormdist(m float64, s float64) *normdist {
	n := &normdist{
		stddev: s,
		mean:   m,
	}
	n.stddevsqpi = s * math.Pow((2*math.Pi), 0.5)
	if s == 1 {
		n.twostddevsq = 2
	} else {
		n.twostddevsq = 2 * (s * s)
	}
	return n
}

func (n *normdist) String() string {
	s := fmt.Sprintf("normdist {mean: %f, stddev: %f}", n.mean, n.stddev)
	return s
}

func (n *normdist) Mean() float64 {
	return n.mean
}

func (n *normdist) Stdev() float64 {
	return n.stddev
}

func (n *normdist) Pdf(x float64) float64 {
	var expon float64
	if n.mean == 0 {
		expon = -(x * x) / n.twostddevsq
	} else {
		expon = -(math.Pow((x - n.mean), 2)) / n.twostddevsq
	}
	probDist := math.Exp(expon) / n.stddevsqpi
	return probDist
}

func (n *normdist) Cdf(x float64) float64 {
	dist := x - n.mean
	if math.Abs(dist) > toomanydev*n.stddev {
		if x < n.mean {
			return 0.0
		} else {
			return 1.0
		}
	}
	errf := Errf(dist / (n.stddev * sqrt2))
	cdf := 0.5 * (1.0 + errf)
	return cdf
}

func Errf(z float64) float64 {
	var t float64
	t = 1.0 / (1.0 + 0.5*math.Abs(z))
	ans := 1 - t*math.Exp(-z*z-1.26551223+
		t*(1.00002368+
			t*(0.37409196+
				t*(0.09678418+
					t*(-0.18628806+
						t*(0.27886807+
							t*(-1.13520398+
								t*(1.48851587+
									t*(-0.82215223+
										t*(0.17087277))))))))))
	if z >= 0 {
		return ans
	}
	return -ans
}

var Stdnorm *normdist = NewNormdist(0.0, 1.0)
