package service

import (
	"avito/train-assignment/app/internal/repository"
	"avito/train-assignment/app/pkg/randutil"
	"errors"

	"github.com/jmoiron/sqlx"
)

// AssignmentService отвечает за выбор кандидатов и запись назначений.
type AssignmentService struct {
	db   *sqlx.DB
	pr   *repository.PRRepo
	revs *repository.ReviewersRepo
	// Доп. репозитории можно добавить по мере необходимости.
}

func NewAssignmentService(db *sqlx.DB, pr *repository.PRRepo, revs *repository.ReviewersRepo) *AssignmentService {
	return &AssignmentService{db: db, pr: pr, revs: revs}
}

// AssignInitial назначает до двух ревьюверов, исключая автора.
// candidates — список активных членов команды автора (без автора).
func (s *AssignmentService) AssignInitial(prID, authorID string, candidates []string) error {
	// Без кандидатов просто возвращаемся — допускаются 0/1 ревьюера по ТЗ.
	if len(candidates) == 0 {
		return nil
	}
	chosen := randutil.PickUpToN(candidates, 2)

	return repository.Tx(s.db, func(tx *sqlx.Tx) error {
		for _, uid := range chosen {
			if err := s.revs.Add(tx, prID, uid); err != nil {
				return err
			}
		}
		// Ограничение «не более 2» соблюдается бизнес‑логикой; доп. enforce можно сделать триггером.
		return nil
	})
}

// Reassign заменяет reviewerOld на случайного активного кандидата из его команды.
// newCandidate должен быть заранее валидирован (активен, не автор и не уже ревьювер).
func (s *AssignmentService) Reassign(prID, reviewerOld, newCandidate string) error {
	return repository.Tx(s.db, func(tx *sqlx.Tx) error {
		if err := s.revs.Remove(tx, prID, reviewerOld); err != nil {
			return err
		}
		if err := s.revs.Add(tx, prID, newCandidate); err != nil {
			return err
		}
		return nil
	})
}

// GuardTwoMax — пример проверки, что число ревьюверов не превышает 2.
func (s *AssignmentService) GuardTwoMax(current []string) error {
	if len(current) > 2 {
		return errors.New("too many reviewers")
	}
	return nil
}
