package main

import (
	"context"
	"log/slog"
	"time"
)

type holmesInvestigator interface {
	Investigate(ctx context.Context, a Alert) (Analysis, error)
}
type patcher interface {
	Generate(ctx context.Context, an Analysis) (*Patch, error)
}
type jiraUpserter interface {
	Upsert(ctx context.Context, a Alert, an Analysis) (IssueKey, error)
}
type prOpener interface {
	OpenPR(ctx context.Context, p Patch, issue IssueKey) (PRLink, error)
}
type discordNotifier interface {
	Notify(ctx context.Context, a Alert, an Analysis, issue IssueKey, pr *PRLink, patch *Patch) error
}

type Pipeline struct {
	holmes  holmesInvestigator
	patch   patcher
	jira    jiraUpserter
	github  prOpener
	discord discordNotifier
	log     *slog.Logger
}

func NewPipeline(h holmesInvestigator, pg patcher, j jiraUpserter, gh prOpener, d discordNotifier, log *slog.Logger) *Pipeline {
	return &Pipeline{holmes: h, patch: pg, jira: j, github: gh, discord: d, log: log}
}

// Handle runs one alert through the pipeline. Each output is independent: a
// failure is logged with the fingerprint and never suppresses later outputs.
func (p *Pipeline) Handle(ctx context.Context, a Alert) error {
	log := p.log.With("fingerprint", a.Fingerprint, "alert", a.Name())

	an, err := p.holmes.Investigate(ctx, a)
	if err != nil {
		// Holmes is terminal: nothing to fan out. Still tell humans so they
		// are not left blind. The per-alert context is usually already dead
		// here (deadline expired mid-investigation), so the notice gets a
		// short-lived context of its own.
		log.Error("holmes investigation failed", "err", err)
		nctx, ncancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer ncancel()
		if derr := p.discord.Notify(nctx, a, Analysis{RootCause: "⚠️ investigation failed: " + err.Error()}, "", nil, nil); derr != nil {
			log.Error("discord failure notice failed", "err", derr)
		}
		return err
	}

	// Structured patch is best-effort; nil is the common case.
	var patch *Patch
	if pt, perr := p.patch.Generate(ctx, an); perr != nil {
		log.Error("patch generation failed", "err", perr)
	} else {
		patch = pt
	}

	var issue IssueKey
	if k, jerr := p.jira.Upsert(ctx, a, an); jerr != nil {
		log.Error("jira upsert failed", "err", jerr)
	} else {
		issue = k
	}

	var prLink *PRLink
	if patch != nil {
		if link, gerr := p.github.OpenPR(ctx, *patch, issue); gerr != nil {
			log.Error("github pr failed", "err", gerr)
		} else {
			prLink = &link
		}
	}

	if derr := p.discord.Notify(ctx, a, an, issue, prLink, patch); derr != nil {
		log.Error("discord notify failed", "err", derr)
	}
	return nil
}
