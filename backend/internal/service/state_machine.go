package service

import "errors"

import "backend/internal/domain"

var ErrInvalidTransition = errors.New("invalid status transition")

// ValidateTransition enforces the PRD state machine matrix:
//
//	open        -> in_progress : agent/admin
//	in_progress -> resolved    : agent/admin
//	resolved    -> closed      : requester/agent/admin
//	resolved    -> open        : requester
//	open        -> closed      : requester/admin
func ValidateTransition(current, target domain.TicketStatus, actor domain.User) error {
	if current == target {
		return nil
	}

	switch current {
	case domain.StatusOpen:
		switch target {
		case domain.StatusInProgress:
			if actor.Role == domain.RoleAgent || actor.Role == domain.RoleAdmin {
				return nil
			}
		case domain.StatusClosed:
			if actor.Role == domain.RoleRequester || actor.Role == domain.RoleAdmin {
				return nil
			}
		}

	case domain.StatusInProgress:
		if target == domain.StatusResolved {
			if actor.Role == domain.RoleAgent || actor.Role == domain.RoleAdmin {
				return nil
			}
		}

	case domain.StatusResolved:
		switch target {
		case domain.StatusClosed:
			return nil
		case domain.StatusOpen:
			if actor.Role == domain.RoleRequester {
				return nil
			}
		}

	case domain.StatusClosed:
		// Terminal state.
	}

	return ErrInvalidTransition
}
