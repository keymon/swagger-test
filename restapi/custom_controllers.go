package restapi

import (
	"sync"
	"sync/atomic"

	errors "github.com/go-openapi/errors"
	middleware "github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/swag"
	"github.com/keymon/swagger-test/models"
	"github.com/keymon/swagger-test/restapi/operations"
	"github.com/keymon/swagger-test/restapi/operations/todos"
)

var items = make(map[int64]*models.Item)
var lastID int64

var itemsLock = &sync.Mutex{}

func newItemID() int64 {
	return atomic.AddInt64(&lastID, 1)
}

func addItem(item *models.Item) error {
	if item == nil {
		return errors.New(500, "item must be present")
	}

	itemsLock.Lock()
	defer itemsLock.Unlock()

	newID := newItemID()
	item.ID = newID
	items[newID] = item

	return nil
}

func updateItem(id int64, item *models.Item) error {
	if item == nil {
		return errors.New(500, "item must be present")
	}

	itemsLock.Lock()
	defer itemsLock.Unlock()

	_, exists := items[id]
	if !exists {
		return errors.NotFound("not found: item %d", id)
	}

	item.ID = id
	items[id] = item
	return nil
}

func deleteItem(id int64) error {
	itemsLock.Lock()
	defer itemsLock.Unlock()

	_, exists := items[id]
	if !exists {
		return errors.NotFound("not found: item %d", id)
	}

	delete(items, id)
	return nil
}

func allItems(since int64, limit int32) (result []*models.Item) {
	result = make([]*models.Item, 0)
	for id, item := range items {
		if len(result) >= int(limit) {
			return
		}
		if since == 0 || id > since {
			result = append(result, item)
		}
	}
	return
}

func customConfigureAPI(api *operations.TodoListAPI) {

	api.TodosAddOneHandler = todos.AddOneHandlerFunc(func(params todos.AddOneParams) middleware.Responder {
		if err := addItem(params.Body); err != nil {
			return todos.NewAddOneDefault(500).WithPayload(&models.Error{Code: 500, Message: swag.String(err.Error())})
		}
		return todos.NewAddOneCreated().WithPayload(params.Body)
	})

	api.TodosDestroyOneHandler = todos.DestroyOneHandlerFunc(func(params todos.DestroyOneParams) middleware.Responder {
		if err := deleteItem(params.ID); err != nil {
			return todos.NewDestroyOneDefault(500).WithPayload(&models.Error{Code: 500, Message: swag.String(err.Error())})
		}
		return todos.NewDestroyOneNoContent()
	})

	api.TodosGetHandler = todos.GetHandlerFunc(func(params todos.GetParams) middleware.Responder {
		mergedParams := todos.NewGetParams()
		mergedParams.Since = swag.Int64(0)
		if params.Since != nil {
			mergedParams.Since = params.Since
		}
		if params.Limit != nil {
			mergedParams.Limit = params.Limit
		}
		return todos.NewGetOK().WithPayload(allItems(*mergedParams.Since, *mergedParams.Limit))
	})

	api.TodosUpdateOneHandler = todos.UpdateOneHandlerFunc(func(params todos.UpdateOneParams) middleware.Responder {
		if err := updateItem(params.ID, params.Body); err != nil {
			return todos.NewUpdateOneDefault(500).WithPayload(&models.Error{Code: 500, Message: swag.String(err.Error())})
		}
		return todos.NewUpdateOneOK().WithPayload(params.Body)
	})

}
