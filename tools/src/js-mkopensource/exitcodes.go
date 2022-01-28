package main

type exitCode int

const (
	DependencyGenerationError exitCode = 1
	MarshallJsonError                  = 2
	WriteError                         = 3
)
