package main

import "github.com/toejough/projctl/internal/escalation"

func escalationList(args escalation.ListArgs) error {
	return escalation.RunList(args)
}

func escalationResolve(args escalation.ResolveArgs) error {
	return escalation.RunResolve(args)
}

func escalationReview(args escalation.ReviewArgs) error {
	return escalation.RunReview(args)
}

func escalationWrite(args escalation.WriteArgs) error {
	return escalation.RunWrite(args)
}
