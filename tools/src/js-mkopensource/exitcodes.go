package main

type exitCode int

const (
	DependencyGenerationError exitCode = iota + 1
	MarshallJsonError
	WriteError
)
