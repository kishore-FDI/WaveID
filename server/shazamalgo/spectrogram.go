package shazamalgo

import "math"

type Complex struct {
	r, i float64
}

func fft(a []Complex) {
	n := len(a)
	if n <= 1 {
		return
	}
	e := make([]Complex, n/2)
	o := make([]Complex, n/2)
	for i := 0; i < n/2; i++ {
		e[i] = a[2*i]
		o[i] = a[2*i+1]
	}
	fft(e)
	fft(o)
	for k := 0; k < n/2; k++ {
		t := -2 * math.Pi * float64(k) / float64(n)
		w := Complex{math.Cos(t), math.Sin(t)}
		wo := Complex{
			w.r*o[k].r - w.i*o[k].i,
			w.r*o[k].i + w.i*o[k].r,
		}
		a[k] = Complex{e[k].r + wo.r, e[k].i + wo.i}
		a[k+n/2] = Complex{e[k].r - wo.r, e[k].i - wo.i}
	}
}

func applyHannFrame(x []float64) {
	n := len(x)
	if n == 0 {
		return
	}
	for i := 0; i < n; i++ {
		w := 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(n-1)))
		x[i] *= w
	}
}

func STFT(x []float64, win, hop int) [][]float64 {
	var spec [][]float64
	for start := 0; start+win <= len(x); start += hop {
		frame := make([]Complex, win)
		tmp := make([]float64, win)
		copy(tmp, x[start:start+win])
		applyHannFrame(tmp)
		for i := 0; i < win; i++ {
			frame[i] = Complex{tmp[i], 0}
		}
		fft(frame)

		mag := make([]float64, win/2)
		for k := 0; k < win/2; k++ {
			mag[k] = math.Hypot(frame[k].r, frame[k].i)
		}
		spec = append(spec, mag)
	}
	return spec
}
