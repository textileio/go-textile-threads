package eventstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"

	dsquery "github.com/ipfs/go-datastore/query"
)

var (
	// ErrInvalidSortingField is returned when a query sorts
	// a result by a non-existent field in the model schema.
	ErrInvalidSortingField = errors.New("sorting field doesn't correspond to instance type")
)

// Query allows to build queries to fetch data from a model.
type Query struct {
	ands []*Criterion
	ors  []*Query
	sort struct {
		field string
		desc  bool
	}
}

// Where starts to create a query condition for a field
func Where(field string) *Criterion {
	return &Criterion{
		fieldPath: field,
	}
}

// And concatenates a new condition in an existing field.
func (q *Query) And(field string) *Criterion {
	return &Criterion{
		fieldPath: field,
		query:     q,
	}
}

// Or concatenates a new condition that is sufficient
// for an instance to satisfy, independant of the current Query.
// Has left-associativity as: (a And b) Or c
func (q *Query) Or(orQuery *Query) *Query {
	q.ors = append(q.ors, orQuery)
	return q
}

// OrderBy specify ascending order for the query results.
// On multiple calls, only the last one is considered.
func (q *Query) OrderBy(field string) *Query {
	q.sort.field = field
	q.sort.desc = false
	return q
}

// OrderByDesc specify descending order for the query results.
// On multiple calls, only the last one is considered.
func (q *Query) OrderByDesc(field string) *Query {
	q.sort.field = field
	q.sort.desc = true
	return q
}

// Find executes a query and store the result in res which should be a slice of
// pointers with the correct model type. If the slice isn't empty, will be emptied.
func (t *Txn) Find(res interface{}, q *Query) error {
	// ToDo: context cancellation? (to call dsr.Close())
	valRes := reflect.ValueOf(res)
	if valRes.Kind() != reflect.Ptr || valRes.Elem().Kind() != reflect.Slice {
		panic("result should be a slice")
	}
	if q == nil {
		q = &Query{}
	}
	dsq := dsquery.Query{
		Prefix: t.model.dsKey.String(),
	}
	dsr, err := t.model.datastore.Query(dsq)
	if err != nil {
		return fmt.Errorf("error when internal query: %v", err)
	}

	resSlice := valRes.Elem()
	resSlice.Set(resSlice.Slice(0, 0))
	// ToDo: also check `res` is slice of *model type*
	var unsorted []reflect.Value
	for {
		res, ok := dsr.NextSync()
		if !ok {
			break
		}
		instance := reflect.New(t.model.valueType.Elem())
		if err = json.Unmarshal(res.Value, instance.Interface()); err != nil {
			return fmt.Errorf("error when unmarshaling query result: %v", err)
		}
		ok, err = q.match(instance)
		if err != nil {
			return fmt.Errorf("error when matching entry with query: %v", err)
		}
		if ok {
			unsorted = append(unsorted, instance)
		}
	}
	if q.sort.field != "" {
		var wrongField, cantCompare bool
		sort.Slice(unsorted, func(i, j int) bool {
			fieldI, err := traverseFieldPath(unsorted[i], q.sort.field)
			if err != nil {
				wrongField = true
				return false
			}
			fieldJ, err := traverseFieldPath(unsorted[j], q.sort.field)
			if err != nil {
				wrongField = true
				return false
			}
			res, err := compare(fieldI.Interface(), fieldJ.Interface())
			if err != nil {
				cantCompare = true
				return false
			}
			if q.sort.desc {
				res *= -1
			}
			return res < 0
		})
		if wrongField {
			return ErrInvalidSortingField
		}
		if cantCompare {
			panic("can't compare while sorting")
		}
	}
	for i := range unsorted {
		resSlice = reflect.Append(resSlice, unsorted[i])
	}
	valRes.Elem().Set(resSlice)
	return nil
}

func (q *Query) match(v reflect.Value) (bool, error) {
	if q == nil {
		panic("query can't be nil")
	}

	andOk := true
	for _, Criterion := range q.ands {
		fieldForMatch, err := traverseFieldPath(v, Criterion.fieldPath)
		if err != nil {
			return false, err
		}
		ok, err := Criterion.match(fieldForMatch)
		if err != nil {
			return false, err
		}
		andOk = andOk && ok
		if !andOk {
			break
		}
	}
	if andOk {
		return true, nil
	}

	for _, q := range q.ors {
		ok, err := q.match(v)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}

	return false, nil
}
