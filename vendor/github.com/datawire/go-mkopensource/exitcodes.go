package main

type exitCode int

const (
	NoError                   exitCode = 0
	DependencyGenerationError exitCode = 1
	InvalidArgumentsError     exitCode = 2
	MarshallJsonError         exitCode = 3
)
