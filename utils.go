package oaiaux

import "math"

// Vector represents an embeddings vector
type Vector []float64

// Length calculates the Euclidean norm/length of this vector.
func (v Vector) Length() float64 {
	result := 0.0
	for _, e := range v {
		result += e * e
	}
	return math.Sqrt(result)
}

// Dot calculates the dot-product of this vector and another.
func (v Vector) Dot(other Vector) float64 {
	result := 0.0
	for i, e := range v {
		result += e * other[i]
	}
	return result
}

// Cosine calculates the cosine-similarity of this vector and another.
func (v Vector) Cosine(other Vector) float64 {
	dot := v.Dot(other)
	cross := v.Length() * other.Length()
	return dot / cross
}
