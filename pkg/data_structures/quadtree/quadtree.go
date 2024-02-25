package quadtree

import rl "github.com/gen2brain/raylib-go/raylib"

type Data struct {
	ID         int
	Boundaries rl.Rectangle
	Value      any
}

type Quadtree struct {
	Capacity int

	Bounds rl.Rectangle

	data map[int]Data

	Regions []*Quadtree
}

func NewQuadtree(bounds rl.Rectangle, capacity int) *Quadtree {
	return &Quadtree{
		Capacity: capacity,
		Bounds:   bounds,
		data:     make(map[int]Data, capacity),
		Regions:  make([]*Quadtree, 0, 4),
	}
}

func (q *Quadtree) Insert(id int, boundaries rl.Rectangle, data any) bool {
	if !rl.CheckCollisionRecs(boundaries, q.Bounds) {
		return false
	}

	// insert in this region given enough space
	if len(q.data) < q.Capacity {
		q.data[id] = Data{
			ID:         id,
			Boundaries: boundaries,
			Value:      data,
		}
		return true
	}

	if len(q.Regions) == 0 {
		q.subdivide()
	}

	// try to insert in subregions
	for _, region := range q.Regions {
		if region.Insert(id, boundaries, data) {
			return true
		}
	}

	// won't fit in any subregion
	// insert in this region and try to rebalance
	q.data[id] = Data{
		ID:         id,
		Boundaries: boundaries,
		Value:      data,
	}
	if !q.rebalance() {
		// couldn't rebalance, don't insert
		delete(q.data, id)
		return false
	}
	return true
}

func (q *Quadtree) Query(rect rl.Rectangle) []Data {
	if !rl.CheckCollisionRecs(q.Bounds, rect) {
		return nil
	}

	var collidedData []Data
	for _, data := range q.data {
		if rl.CheckCollisionRecs(data.Boundaries, rect) {
			collidedData = append(collidedData, data)
		}
	}

	for _, region := range q.Regions {
		collidedData = append(collidedData, region.Query(rect)...)
	}

	return collidedData
}

func (q *Quadtree) Clear() {
	clear(q.data)
	for _, region := range q.Regions {
		region.Clear()
	}

}

func (q *Quadtree) subdivide() {
	width := q.Bounds.Width / 2
	height := q.Bounds.Height / 2
	// north west
	q.Regions = append(q.Regions, NewQuadtree(rl.Rectangle{
		X:      q.Bounds.X,
		Y:      q.Bounds.Y,
		Width:  width,
		Height: height,
	}, q.Capacity))
	// north east
	q.Regions = append(q.Regions, NewQuadtree(rl.Rectangle{
		X:      q.Bounds.X + q.Bounds.Width/2,
		Y:      q.Bounds.Y,
		Width:  width,
		Height: height,
	}, q.Capacity))
	// south west
	q.Regions = append(q.Regions, NewQuadtree(rl.Rectangle{
		X:      q.Bounds.X,
		Y:      q.Bounds.Y + q.Bounds.Height/2,
		Width:  width,
		Height: height,
	}, q.Capacity))
	// south east
	q.Regions = append(q.Regions, NewQuadtree(rl.Rectangle{
		X:      q.Bounds.X + q.Bounds.Width/2,
		Y:      q.Bounds.Y + q.Bounds.Height/2,
		Width:  width,
		Height: height,
	}, q.Capacity))
}

func (q *Quadtree) rebalance() bool {
	if len(q.data) <= q.Capacity {
		return true
	}

	for id, data := range q.data {
		// try to move to subregion
		for _, region := range q.Regions {
			if region.Insert(id, data.Boundaries, data.Value) {
				delete(q.data, id)
				break
			}
		}
	}

	return len(q.data) <= q.Capacity
}
