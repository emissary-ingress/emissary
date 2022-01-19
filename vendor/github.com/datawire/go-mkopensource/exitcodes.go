package main

type exitCode int

const (
	NoError exitCode = iota
	DependencyGenerationError
	InvalidArgumentsError
	MarshallJsonError
)
