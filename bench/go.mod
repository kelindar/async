module github.com/kelindar/async/bench

go 1.24.0

toolchain go1.24.4

require (
	github.com/kelindar/async v0.0.0
	github.com/kelindar/bench v0.3.2
)

require gonum.org/v1/gonum v0.17.0 // indirect

replace github.com/kelindar/async => ../
