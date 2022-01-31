package main

type exitCode int

const (
	DependencyGenerationError exitCode = 1
	InvalidArgumentsError     exitCode = 2
	MarshallJsonError         exitCode = 3
	WriteError                exitCode = 4
)
