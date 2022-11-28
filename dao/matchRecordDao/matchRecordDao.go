package matchRecordDao

import (
	"database/sql"
	"errors"

	"github.com/paper-trade-chatbot/be-common/pagination"
	"github.com/paper-trade-chatbot/be-match/models/dbModels"
	"github.com/paper-trade-chatbot/be-proto/general"
	"github.com/shopspring/decimal"

	"gorm.io/gorm"
)

const table = "match_record"

// QueryModel set query condition, used by queryChain()
type QueryModel struct {
	ID uint64
}

type UpdateModel struct {
	MatchStatus  *dbModels.MatchStatus
	UnitPrice    *decimal.NullDecimal
	RollbackerID *sql.NullInt64
	RollbackedAt *sql.NullTime
	Remark       *sql.NullString
}

// New a row
func New(db *gorm.DB, model *dbModels.MatchRecordModel) (int, error) {

	err := db.Table(table).
		Create(model).Error

	if err != nil {
		return 0, err
	}
	return 1, nil
}

// New rows
func News(db *gorm.DB, m []*dbModels.MatchRecordModel) (int, error) {

	err := db.Transaction(func(tx *gorm.DB) error {

		err := tx.Table(table).
			CreateInBatches(m, 3000).Error

		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return 0, err
	}

	return len(m), nil
}

// Get return a record as raw-data-form
func Get(tx *gorm.DB, query *QueryModel) (*dbModels.MatchRecordModel, error) {

	result := &dbModels.MatchRecordModel{}
	err := tx.Table(table).
		Scopes(queryChain(query)).
		Scan(result).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Gets return records as raw-data-form
func Gets(tx *gorm.DB, query *QueryModel) ([]dbModels.MatchRecordModel, error) {
	result := make([]dbModels.MatchRecordModel, 0)
	err := tx.Table(table).
		Scopes(queryChain(query)).
		Scan(&result).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return []dbModels.MatchRecordModel{}, nil
	}

	if err != nil {
		return nil, err
	}

	return result, nil
}

func GetsWithPagination(tx *gorm.DB, query *QueryModel, paginate *general.Pagination) ([]dbModels.MatchRecordModel, *general.PaginationInfo, error) {

	var rows []dbModels.MatchRecordModel
	var count int64 = 0
	err := tx.Table(table).
		Scopes(queryChain(query)).
		Count(&count).
		Scopes(paginateChain(paginate)).
		Scan(&rows).Error

	offset, _ := pagination.GetOffsetAndLimit(paginate)
	paginationInfo := pagination.SetPaginationDto(paginate.Page, paginate.PageSize, int32(count), int32(offset))

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return []dbModels.MatchRecordModel{}, paginationInfo, nil
	}

	if err != nil {
		return []dbModels.MatchRecordModel{}, nil, err
	}

	return rows, paginationInfo, nil
}

// Gets return records as raw-data-form
func Modify(tx *gorm.DB, model *dbModels.MatchRecordModel, update *UpdateModel) error {
	attrs := map[string]interface{}{}
	if update.MatchStatus != nil {
		attrs["match_status"] = *update.MatchStatus
	}

	err := tx.Table(table).
		Model(dbModels.MatchRecordModel{}).
		Where(table+".id = ?", model.ID).
		Updates(attrs).Error

	return err
}

func queryChain(query *QueryModel) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.
			Scopes(idEqualScope(query.ID))
	}
}

func paginateChain(paginate *general.Pagination) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		offset, limit := pagination.GetOffsetAndLimit(paginate)
		return db.
			Scopes(offsetScope(offset)).
			Scopes(limitScope(limit))

	}
}

func idEqualScope(id uint64) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if id != 0 {
			return db.Where(table+".id = ?", id)
		}
		return db
	}
}

func limitScope(limit int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if limit > 0 {
			return db.Limit(limit)
		}
		return db
	}
}

func offsetScope(offset int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if offset > 0 {
			return db.Limit(offset)
		}
		return db
	}
}
