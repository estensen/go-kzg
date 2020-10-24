// Experimental translation to Go.
// Original:
// - https://github.com/ethereum/research/blob/master/mimc_stark/fft.py
// - https://github.com/ethereum/research/blob/master/mimc_stark/recovery.py

package go_verkle

import (
	"fmt"
	"strings"
)

var scale2RootOfUnity []Big

var ZERO, ONE, TWO Big
var MODULUS_MINUS1, MODULUS_MINUS1_DIV2, MODULUS_MINUS2 Big

func initGlobals() {
	bigNumHelper := func(v string) (out Big) {
		bigNum(&out, v)
		return
	}
	scale2RootOfUnity = []Big{
		/* k=0          r=1          */ bigNumHelper("1"),
		/* k=1          r=2          */ bigNumHelper("52435875175126190479447740508185965837690552500527637822603658699938581184512"),
		/* k=2          r=4          */ bigNumHelper("3465144826073652318776269530687742778270252468765361963008"),
		/* k=3          r=8          */ bigNumHelper("23674694431658770659612952115660802947967373701506253797663184111817857449850"),
		/* k=4          r=16         */ bigNumHelper("14788168760825820622209131888203028446852016562542525606630160374691593895118"),
		/* k=5          r=32         */ bigNumHelper("36581797046584068049060372878520385032448812009597153775348195406694427778894"),
		/* k=6          r=64         */ bigNumHelper("31519469946562159605140591558550197856588417350474800936898404023113662197331"),
		/* k=7          r=128        */ bigNumHelper("47309214877430199588914062438791732591241783999377560080318349803002842391998"),
		/* k=8          r=256        */ bigNumHelper("36007022166693598376559747923784822035233416720563672082740011604939309541707"),
		/* k=9          r=512        */ bigNumHelper("4214636447306890335450803789410475782380792963881561516561680164772024173390"),
		/* k=10         r=1024       */ bigNumHelper("22781213702924172180523978385542388841346373992886390990881355510284839737428"),
		/* k=11         r=2048       */ bigNumHelper("49307615728544765012166121802278658070711169839041683575071795236746050763237"),
		/* k=12         r=4096       */ bigNumHelper("39033254847818212395286706435128746857159659164139250548781411570340225835782"),
		/* k=13         r=8192       */ bigNumHelper("32731401973776920074999878620293785439674386180695720638377027142500196583783"),
		/* k=14         r=16384      */ bigNumHelper("39072540533732477250409069030641316533649120504872707460480262653418090977761"),
		/* k=15         r=32768      */ bigNumHelper("22872204467218851938836547481240843888453165451755431061227190987689039608686"),
		/* k=16         r=65536      */ bigNumHelper("15076889834420168339092859836519192632846122361203618639585008852351569017005"),
		/* k=17         r=131072     */ bigNumHelper("15495926509001846844474268026226183818445427694968626800913907911890390421264"),
		/* k=18         r=262144     */ bigNumHelper("20439484849038267462774237595151440867617792718791690563928621375157525968123"),
		/* k=19         r=524288     */ bigNumHelper("37115000097562964541269718788523040559386243094666416358585267518228781043101"),
	}

	asBig(&ZERO, 0)
	asBig(&ONE, 1)
	asBig(&TWO, 2)

	subModBig(&MODULUS_MINUS1, &ZERO, &ONE)
	divModBig(&MODULUS_MINUS1_DIV2, &MODULUS_MINUS1, &TWO)
	subModBig(&MODULUS_MINUS2, &ZERO, &TWO)
}

// Expands the power circle for a given root of unity to WIDTH+1 values.
// The first entry will be 1, the last entry won't be 1 (omitted, unlike the python implementation)
func expandRootOfUnity(rootOfUnity *Big) []Big {
	rootz := make([]Big, 2)
	rootz[0] = ONE // some unused number in py code
	rootz[1] = *rootOfUnity
	for i := 1; !equalOne(&rootz[i]); {
		rootz = append(rootz, Big{})
		this := &rootz[i]
		i++
		mulModBig(&rootz[i], this, rootOfUnity)
	}
	return rootz
}

type FFTSettings struct {
	scale uint8
	width uint64
	// the generator used to get all roots of unity
	rootOfUnity          *Big
	expandedRootsOfUnity []Big
	reverseRootsOfUnity  []Big
}

func NewFFTSettings(scale uint8) *FFTSettings {
	width := uint64(1) << scale
	root := &scale2RootOfUnity[scale]
	rootz := expandRootOfUnity(&scale2RootOfUnity[scale])
	// reverse roots of unity
	rootzReverse := make([]Big, len(rootz), len(rootz))
	copy(rootzReverse, rootz)
	for i, j := uint64(0), uint64(len(rootz)-1); i < j; i, j = i+1, j-1 {
		rootzReverse[i], rootzReverse[j] = rootzReverse[j], rootzReverse[i]
	}
	return &FFTSettings{
		scale:                scale,
		width:                width,
		rootOfUnity:          root,
		expandedRootsOfUnity: rootz,
		reverseRootsOfUnity:  rootzReverse,
	}
}

func (fs *FFTSettings) zPoly(positions []uint) []Big {
	return fs._zPoly(positions, 1)
}

func debugBigPtrs(msg string, values []*Big) {
	var out strings.Builder
	out.WriteString("---")
	out.WriteString(msg)
	out.WriteString("---\n")
	for i := range values {
		out.WriteString(fmt.Sprintf("#%4d: %s\n", i, bigStr(values[i])))
	}
	fmt.Println(out.String())
}

func debugBigs(msg string, values []Big) {
	var out strings.Builder
	out.WriteString("---")
	out.WriteString(msg)
	out.WriteString("---\n")
	for i := range values {
		out.WriteString(fmt.Sprintf("#%4d: %s\n", i, bigStr(&values[i])))
	}
	fmt.Println(out.String())
}

func debugBigsOffsetStride(msg string, values []Big, offset uint, stride uint) {
	var out strings.Builder
	out.WriteString("---")
	out.WriteString(msg)
	out.WriteString("---\n")
	j := uint(0)
	for i := offset; i < uint(len(values)); i += stride {
		out.WriteString(fmt.Sprintf("#%4d: %s\n", j, bigStr(&values[i])))
		j++
	}
	fmt.Println(out.String())
}

type Config struct {
	WIDTH int
}

func (fs *FFTSettings) simpleFT(vals []Big, valsOffset uint, valsStride uint, rootsOfUnity []Big, rootsOfUnityStride uint, out []Big) {
	l := uint(len(out))
	var v Big
	for i := uint(0); i < l; i++ {
		last := ZERO

		for j := uint(0); j < l; j++ {
			jv := &vals[valsOffset+j*valsStride]
			r := &rootsOfUnity[((i*j)%l)*rootsOfUnityStride]
			mulModBig(&v, jv, r) // TODO lookup could be optimized
			addModBig(&last, &last, &v)
		}
		out[i] = last
	}
}

func (fs *FFTSettings) _fft(vals []Big, valsOffset uint, valsStride uint, rootsOfUnity []Big, rootsOfUnityStride uint, out []Big) {
	if len(out) <= 4 { // if the value count is small, run the unoptimized version instead. // TODO tune threshold.
		fs.simpleFT(vals, valsOffset, valsStride, rootsOfUnity, rootsOfUnityStride, out)
		return
	}

	half := uint(len(out)) >> 1
	// L will be the left half of out
	fs._fft(vals, valsOffset, valsStride<<1, rootsOfUnity, rootsOfUnityStride<<1, out[:half])
	// R will be the right half of out
	fs._fft(vals, valsOffset+valsStride, valsStride<<1, rootsOfUnity, rootsOfUnityStride<<1, out[half:]) // just take even again

	var yTimesRoot Big
	var x, y Big
	for i := uint(0); i < half; i++ {
		// temporary copies, so that writing to output doesn't conflict with input
		copyBigNum(&x, &out[i])
		copyBigNum(&y, &out[i+half])
		root := &rootsOfUnity[i*rootsOfUnityStride]
		mulModBig(&yTimesRoot, &y, root)
		addModBig(&out[i], &x, &yTimesRoot)
		subModBig(&out[i+half], &x, &yTimesRoot)
	}
}

func (fs *FFTSettings) FFT(vals []Big, inv bool) ([]Big, error) {
	if len(fs.expandedRootsOfUnity) < len(vals) {
		return nil, fmt.Errorf("got %d values but only have %d roots of unity", len(vals), len(fs.expandedRootsOfUnity))
	}
	// We make a copy so we can mutate it during the work.
	valsCopy := make([]Big, fs.width, fs.width)
	copy(valsCopy, vals)
	// Fill in vals with zeroes if needed
	for i := uint64(len(vals)); i < fs.width; i++ {
		valsCopy[i] = ZERO
	}
	if inv {
		var invLen Big
		asBig(&invLen, uint64(len(vals)))
		invModBig(&invLen, &invLen)
		rootz := fs.reverseRootsOfUnity

		out := make([]Big, fs.width, fs.width)
		fs._fft(valsCopy, 0, 1, rootz[:len(rootz)-1], 1, out)
		for i := 0; i < len(out); i++ {
			mulModBig(&out[i], &out[i], &invLen)
		}
		return out, nil
	} else {
		out := make([]Big, fs.width, fs.width)
		rootz := fs.expandedRootsOfUnity
		// Regular FFT
		fs._fft(valsCopy, 0, 1, rootz[:len(rootz)-1], 1, out)
		return out, nil
	}
}

func (fs *FFTSettings) mulPolys(a []Big, b []Big, rootsOfUnityStride uint) []Big {
	// pad a and b to match roots of unity
	size := fs.width / uint64(rootsOfUnityStride)
	aVals := make([]Big, size, size)
	bVals := make([]Big, size, size)
	for i := 0; i < len(a); i++ {
		aVals[i] = a[i]
	}
	for i := len(a); i < len(aVals); i++ {
		aVals[i] = ZERO
	}
	for i := 0; i < len(b); i++ {
		bVals[i] = b[i]
	}
	for i := len(b); i < len(bVals); i++ {
		bVals[i] = ZERO
	}
	rootz := fs.expandedRootsOfUnity
	// Get FFT of a and b
	x1 := make([]Big, len(aVals), len(aVals))
	fs._fft(aVals, 0, 1, rootz[:len(rootz)-1], rootsOfUnityStride, x1)

	x2 := make([]Big, len(bVals), len(bVals))
	fs._fft(bVals, 0, 1, rootz[:len(rootz)-1], rootsOfUnityStride, x2)

	// multiply the two. Hack: store results in x1
	var tmp Big
	for i := 0; i < len(x1); i++ {
		copyBigNum(&tmp, &x1[i])
		mulModBig(&x1[i], &tmp, &x2[i])
	}
	revRootz := fs.reverseRootsOfUnity

	out := make([]Big, len(x1), len(x1))
	// compute the FFT of the multiplied values.
	fs._fft(x1, 0, 1, revRootz[:len(revRootz)-1], rootsOfUnityStride, out)
	return out
}

// Calculates modular inverses [1/values[0], 1/values[1] ...]
func multiInv(values []Big) []Big {
	partials := make([]Big, len(values)+1, len(values)+1)
	partials[0] = values[0]
	for i := 0; i < len(values); i++ {
		mulModBig(&partials[i+1], &partials[i], &values[i])
	}
	var inv Big
	var tmp Big
	invModBig(&inv, &partials[len(partials)-1])
	outputs := make([]Big, len(values), len(values))
	for i := len(values); i > 0; i-- {
		mulModBig(&outputs[i-1], &partials[i-1], &inv)
		copyBigNum(&tmp, &inv)
		mulModBig(&inv, &tmp, &values[i-1])
	}
	return outputs
}

// Generates q(x) = poly(k * x)
func pOfKX(poly []Big, k *Big) []Big {
	out := make([]Big, len(poly), len(poly))
	powerOfK := ONE
	var tmp Big
	for i := range poly {
		mulModBig(&out[i], &poly[i], &powerOfK)
		copyBigNum(&tmp, &powerOfK)
		mulModBig(&powerOfK, &tmp, k)
	}
	return out
}

func inefficientOddEvenDiv2(positions []uint) (even []uint, odd []uint) { // TODO optimize away
	for _, p := range positions {
		if p&1 == 0 {
			even = append(even, p>>1)
		} else {
			odd = append(odd, p>>1)
		}
	}
	return
}

// Return (x - root**positions[0]) * (x - root**positions[1]) * ...
// possibly with a constant factor offset
func (fs *FFTSettings) _zPoly(positions []uint, rootsOfUnityStride uint) []Big {
	// If there are not more than 4 positions, use the naive
	// O(n^2) algorithm as it is faster
	if len(positions) <= 4 {
		/*
		   root = [1]
		   for pos in positions:
		       x = roots_of_unity[pos]
		       root.insert(0, 0)
		       for j in range(len(root)-1):
		           root[j] -= root[j+1] * x
		   return [x % modulus for x in root]
		*/
		root := make([]Big, len(positions)+1, len(positions)+1)
		root[0] = ONE
		i := 1
		var v Big
		var tmp Big
		for _, pos := range positions {
			x := &fs.expandedRootsOfUnity[pos*rootsOfUnityStride]
			root[i] = ZERO
			for j := i; j >= 1; j-- {
				mulModBig(&v, &root[j-1], x)
				copyBigNum(&tmp, &root[j])
				subModBig(&root[j], &tmp, &v)
			}
			i++
		}
		// We did the reverse representation of 'root' as the python code, to not insert at the start all the time.
		// Now turn it back around.
		for i, j := 0, len(root)-1; i < j; i, j = i+1, j-1 {
			root[i], root[j] = root[j], root[i]
		}
		return root
	}
	// Recursively find the zpoly for even indices and odd
	// indices, operating over a half-size subgroup in each case
	evenPositions, oddPositions := inefficientOddEvenDiv2(positions)
	left := fs._zPoly(evenPositions, rootsOfUnityStride<<1)
	right := fs._zPoly(oddPositions, rootsOfUnityStride<<1)
	invRoot := &fs.expandedRootsOfUnity[uint(len(fs.expandedRootsOfUnity))-1-rootsOfUnityStride]
	// Offset the result for the odd indices, and combine the two
	out := fs.mulPolys(left, pOfKX(right, invRoot), rootsOfUnityStride)
	// Deal with the special case where mul_polys returns zero
	// when it should return x ^ (2 ** k) - 1
	isZero := true
	for i := range out {
		if !equalZero(&out[i]) {
			isZero = false
			break
		}
	}
	if isZero {
		// TODO: it's [1] + [0] * (len(out) - 1) + [modulus - 1] in python, but strange it's 1 larger than out
		out[0] = ONE
		for i := 1; i < len(out); i++ {
			out[i] = ZERO
		}
		last := MODULUS_MINUS1
		out = append(out, last)
		return out
	} else {
		return out
	}
}

// TODO test unhappy case
const maxRecoverAttempts = 10

func (fs *FFTSettings) ErasureCodeRecover(vals []*Big) ([]Big, error) {
	// Generate the polynomial that is zero at the roots of unity
	// corresponding to the indices where vals[i] is None
	positions := make([]uint, 0, len(vals))
	for i := uint(0); i < uint(len(vals)); i++ {
		if vals[i] == nil {
			positions = append(positions, i)
		}
	}
	z := fs.zPoly(positions)
	//debugBigs("z", z)
	zVals, err := fs.FFT(z, false)
	if err != nil {
		return nil, err
	}
	//debugBigs("zvals", zVals)

	// Pointwise-multiply (vals filling in zero at missing spots) * z
	// By construction, this equals vals * z
	pTimesZVals := make([]Big, len(vals), len(vals))
	for i := uint(0); i < uint(len(vals)); i++ {
		if vals[i] == nil {
			// 0 * zVals[i] == 0
			pTimesZVals[i] = ZERO
		} else {
			mulModBig(&pTimesZVals[i], vals[i], &zVals[i])
		}
	}
	//debugBigs("p_times_z_vals", pTimesZVals)
	pTimesZ, err := fs.FFT(pTimesZVals, true)
	if err != nil {
		return nil, err
	}
	//debugBigs("p_times_z", pTimesZ)

	// Keep choosing k values until the algorithm does not fail
	// Check only with primitive roots of unity
	attempts := 0
	var kBig Big
	var tmp Big
	for k := uint64(2); attempts < maxRecoverAttempts; k++ {
		asBig(&kBig, k)
		// // TODO: check this, translation of 'if pow(k, (modulus - 1) // 2, modulus) == 1:'
		sqrModBig(&tmp, &kBig)
		if equalOne(&tmp) {
			continue
		}
		var invk Big
		invModBig(&invk, &kBig)
		// Convert p_times_z(x) and z(x) into new polynomials
		// q1(x) = p_times_z(k*x) and q2(x) = z(k*x)
		// These are likely to not be 0 at any of the evaluation points.
		pTimesZOfKX := pOfKX(pTimesZ, &kBig)
		//debugBigs("p_times_z_of_kx", pTimesZOfKX)
		pTimesZOfKXVals, err := fs.FFT(pTimesZOfKX, false)
		if err != nil {
			return nil, err
		}
		//debugBigs("p_times_z_of_kx_vals", pTimesZOfKXVals)
		zOfKX := pOfKX(z, &kBig)
		//debugBigs("z_of_kx", zOfKX)
		zOfKXVals, err := fs.FFT(zOfKX, false)
		if err != nil {
			return nil, err
		}
		//debugBigs("z_of_kx_vals", zOfKXVals)

		// Compute q1(x) / q2(x) = p(k*x)
		invZOfKXVals := multiInv(zOfKXVals)
		//debugBigs("inv_z_of_kv_vals", invZOfKXVals)
		pOfKxVals := make([]Big, len(pTimesZOfKXVals), len(pTimesZOfKXVals))
		for i := 0; i < len(pOfKxVals); i++ {
			mulModBig(&pOfKxVals[i], &pTimesZOfKXVals[i], &invZOfKXVals[i])
		}
		//debugBigs("p_of_kx_vals", pOfKxVals)
		pOfKx, err := fs.FFT(pOfKxVals, true)
		if err != nil {
			return nil, err
		}
		//debugBigs("p_of_kx", pOfKx)

		// Given q3(x) = p(k*x), recover p(x)
		pOfX := make([]Big, len(pOfKx), len(pOfKx))
		if len(pOfKx) >= 1 {
			pOfX[0] = pOfKx[0]
		}
		if len(pOfKx) >= 2 {
			mulModBig(&pOfX[1], &pOfKx[1], &invk)
			invKPowI := invk
			for i := 2; i < len(pOfKx); i++ {
				copyBigNum(&tmp, &invKPowI)
				mulModBig(&invKPowI, &tmp, &invk)
				mulModBig(&pOfX[i], &pOfKx[i], &invKPowI)
			}
		}
		output, err := fs.FFT(pOfX, false)
		if err != nil {
			return nil, err
		}

		// Check that the output matches the input
		success := true
		for i, inpd := range vals {
			if inpd == nil {
				continue
			}
			if !equalBig(inpd, &output[i]) {
				success = false
				break
			}
		}

		if !success {
			attempts += 1
			continue
		}
		// Output the evaluations if all good
		return output, nil
	}
	return nil, fmt.Errorf("max attempts reached: %d", attempts)
}
