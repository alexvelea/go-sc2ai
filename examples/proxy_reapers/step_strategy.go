package main

import (
	"log"
)

type Strategy interface {
	Strategy()
}

type stepStrategy struct {
	cStep      int
	step       int
	finishStep int
}

func (s *stepStrategy) advance() {
	s.step += 1
	log.Printf("moving opener to step %v", s.step)
}

func (s *stepStrategy) now() bool {
	a := s.cStep
	s.cStep += 1
	if s.finishStep < s.cStep {
		s.finishStep = s.cStep
	}
	return s.step == a
}

func (s *stepStrategy) tooEarly() bool {
	return s.step < s.cStep
}

func (s *stepStrategy) finished() bool {
	s.cStep = 0
	return s.step != 0 && s.step == s.finishStep
}
