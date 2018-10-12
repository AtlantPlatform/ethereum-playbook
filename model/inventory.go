package model

type Inventory map[string]InventorySpec

func (inventory *Inventory) Validate(ctx AppContext, spec *Spec) bool {
	return true
}

type InventorySpec []string
