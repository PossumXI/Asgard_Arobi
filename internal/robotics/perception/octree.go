// Package perception provides spatial indexing via Octree for fast 360-degree queries.
//
// Copyright 2026 Arobi. All Rights Reserved.
package perception

import (
	"sync"
)

const maxObjectsPerNode = 8

// OctreeNode represents a node in the octree spatial index
type OctreeNode struct {
	Center   Vector3
	HalfSize float64
	Objects  []*TrackedObject
	Children [8]*OctreeNode
	IsLeaf   bool
}

// Octree provides spatial indexing for fast object queries
type Octree struct {
	mu   sync.RWMutex
	root *OctreeNode
}

// NewOctree creates a new octree with given center and size
func NewOctree(center Vector3, halfSize float64) *Octree {
	return &Octree{
		root: &OctreeNode{
			Center:   center,
			HalfSize: halfSize,
			Objects:  make([]*TrackedObject, 0, maxObjectsPerNode),
			IsLeaf:   true,
		},
	}
}

// Insert adds an object to the octree
func (o *Octree) Insert(obj *TrackedObject) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.insertNode(o.root, obj)
}

func (o *Octree) insertNode(node *OctreeNode, obj *TrackedObject) {
	if node.IsLeaf {
		if len(node.Objects) < maxObjectsPerNode {
			node.Objects = append(node.Objects, obj)
			return
		}
		// Split the node
		o.splitNode(node)
	}

	// Find the appropriate child
	octant := o.getOctant(node, obj.Position)
	if node.Children[octant] == nil {
		node.Children[octant] = &OctreeNode{
			Center:   o.getChildCenter(node, octant),
			HalfSize: node.HalfSize / 2,
			Objects:  make([]*TrackedObject, 0, maxObjectsPerNode),
			IsLeaf:   true,
		}
	}
	o.insertNode(node.Children[octant], obj)
}

// splitNode divides a leaf node into 8 children
func (o *Octree) splitNode(node *OctreeNode) {
	node.IsLeaf = false
	objects := node.Objects
	node.Objects = nil

	for _, obj := range objects {
		octant := o.getOctant(node, obj.Position)
		if node.Children[octant] == nil {
			node.Children[octant] = &OctreeNode{
				Center:   o.getChildCenter(node, octant),
				HalfSize: node.HalfSize / 2,
				Objects:  make([]*TrackedObject, 0, maxObjectsPerNode),
				IsLeaf:   true,
			}
		}
		o.insertNode(node.Children[octant], obj)
	}
}

// getOctant returns the octant index (0-7) for a position
func (o *Octree) getOctant(node *OctreeNode, pos Vector3) int {
	octant := 0
	if pos.X >= node.Center.X {
		octant |= 1
	}
	if pos.Y >= node.Center.Y {
		octant |= 2
	}
	if pos.Z >= node.Center.Z {
		octant |= 4
	}
	return octant
}

// getChildCenter returns the center of a child octant
func (o *Octree) getChildCenter(node *OctreeNode, octant int) Vector3 {
	offset := node.HalfSize / 2
	center := node.Center

	if octant&1 != 0 {
		center.X += offset
	} else {
		center.X -= offset
	}
	if octant&2 != 0 {
		center.Y += offset
	} else {
		center.Y -= offset
	}
	if octant&4 != 0 {
		center.Z += offset
	} else {
		center.Z -= offset
	}

	return center
}

// Remove removes an object from the octree
func (o *Octree) Remove(obj *TrackedObject) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.removeNode(o.root, obj)
}

func (o *Octree) removeNode(node *OctreeNode, obj *TrackedObject) bool {
	if node == nil {
		return false
	}

	if node.IsLeaf {
		for i, stored := range node.Objects {
			if stored.ID == obj.ID {
				// Remove by swapping with last element
				node.Objects[i] = node.Objects[len(node.Objects)-1]
				node.Objects = node.Objects[:len(node.Objects)-1]
				return true
			}
		}
		return false
	}

	octant := o.getOctant(node, obj.Position)
	return o.removeNode(node.Children[octant], obj)
}

// QueryRadius returns all objects within radius of a center point
func (o *Octree) QueryRadius(center Vector3, radius float64) []*TrackedObject {
	o.mu.RLock()
	defer o.mu.RUnlock()

	result := make([]*TrackedObject, 0)
	o.queryRadiusNode(o.root, center, radius, &result)
	return result
}

func (o *Octree) queryRadiusNode(node *OctreeNode, center Vector3, radius float64, result *[]*TrackedObject) {
	if node == nil {
		return
	}

	// Check if this node's bounding box intersects the query sphere
	if !o.sphereIntersectsBox(center, radius, node) {
		return
	}

	if node.IsLeaf {
		for _, obj := range node.Objects {
			if center.Distance(obj.Position) <= radius {
				*result = append(*result, obj)
			}
		}
		return
	}

	// Recurse into children
	for _, child := range node.Children {
		o.queryRadiusNode(child, center, radius, result)
	}
}

// sphereIntersectsBox checks if a sphere intersects an AABB
func (o *Octree) sphereIntersectsBox(center Vector3, radius float64, node *OctreeNode) bool {
	// Find the closest point on the box to the sphere center
	closestX := clamp(center.X, node.Center.X-node.HalfSize, node.Center.X+node.HalfSize)
	closestY := clamp(center.Y, node.Center.Y-node.HalfSize, node.Center.Y+node.HalfSize)
	closestZ := clamp(center.Z, node.Center.Z-node.HalfSize, node.Center.Z+node.HalfSize)

	// Calculate distance from closest point to sphere center
	closest := Vector3{X: closestX, Y: closestY, Z: closestZ}
	return center.Distance(closest) <= radius
}

// QueryBox returns all objects within a bounding box
func (o *Octree) QueryBox(min, max Vector3) []*TrackedObject {
	o.mu.RLock()
	defer o.mu.RUnlock()

	result := make([]*TrackedObject, 0)
	o.queryBoxNode(o.root, min, max, &result)
	return result
}

func (o *Octree) queryBoxNode(node *OctreeNode, min, max Vector3, result *[]*TrackedObject) {
	if node == nil {
		return
	}

	// Check if this node's bounding box intersects the query box
	if !o.boxIntersectsBox(min, max, node) {
		return
	}

	if node.IsLeaf {
		for _, obj := range node.Objects {
			if obj.Position.X >= min.X && obj.Position.X <= max.X &&
				obj.Position.Y >= min.Y && obj.Position.Y <= max.Y &&
				obj.Position.Z >= min.Z && obj.Position.Z <= max.Z {
				*result = append(*result, obj)
			}
		}
		return
	}

	// Recurse into children
	for _, child := range node.Children {
		o.queryBoxNode(child, min, max, result)
	}
}

// boxIntersectsBox checks if two AABBs intersect
func (o *Octree) boxIntersectsBox(min, max Vector3, node *OctreeNode) bool {
	nodeMin := Vector3{
		X: node.Center.X - node.HalfSize,
		Y: node.Center.Y - node.HalfSize,
		Z: node.Center.Z - node.HalfSize,
	}
	nodeMax := Vector3{
		X: node.Center.X + node.HalfSize,
		Y: node.Center.Y + node.HalfSize,
		Z: node.Center.Z + node.HalfSize,
	}

	return min.X <= nodeMax.X && max.X >= nodeMin.X &&
		min.Y <= nodeMax.Y && max.Y >= nodeMin.Y &&
		min.Z <= nodeMax.Z && max.Z >= nodeMin.Z
}

// NearestNeighbor finds the nearest object to a point
func (o *Octree) NearestNeighbor(point Vector3) *TrackedObject {
	o.mu.RLock()
	defer o.mu.RUnlock()

	var nearest *TrackedObject
	nearestDist := float64(1e18)

	o.nearestNeighborNode(o.root, point, &nearest, &nearestDist)
	return nearest
}

func (o *Octree) nearestNeighborNode(node *OctreeNode, point Vector3, nearest **TrackedObject, nearestDist *float64) {
	if node == nil {
		return
	}

	// Check if this node could contain a closer point
	minDist := o.minDistToBox(point, node)
	if minDist > *nearestDist {
		return
	}

	if node.IsLeaf {
		for _, obj := range node.Objects {
			dist := point.Distance(obj.Position)
			if dist < *nearestDist {
				*nearestDist = dist
				*nearest = obj
			}
		}
		return
	}

	// Recurse into children, prioritizing by distance
	for _, child := range node.Children {
		o.nearestNeighborNode(child, point, nearest, nearestDist)
	}
}

// minDistToBox returns minimum distance from point to box
func (o *Octree) minDistToBox(point Vector3, node *OctreeNode) float64 {
	closestX := clamp(point.X, node.Center.X-node.HalfSize, node.Center.X+node.HalfSize)
	closestY := clamp(point.Y, node.Center.Y-node.HalfSize, node.Center.Y+node.HalfSize)
	closestZ := clamp(point.Z, node.Center.Z-node.HalfSize, node.Center.Z+node.HalfSize)

	closest := Vector3{X: closestX, Y: closestY, Z: closestZ}
	return point.Distance(closest)
}

// GetAllObjects returns all objects in the octree
func (o *Octree) GetAllObjects() []*TrackedObject {
	o.mu.RLock()
	defer o.mu.RUnlock()

	result := make([]*TrackedObject, 0)
	o.getAllObjectsNode(o.root, &result)
	return result
}

func (o *Octree) getAllObjectsNode(node *OctreeNode, result *[]*TrackedObject) {
	if node == nil {
		return
	}

	if node.IsLeaf {
		*result = append(*result, node.Objects...)
		return
	}

	for _, child := range node.Children {
		o.getAllObjectsNode(child, result)
	}
}

// Clear removes all objects from the octree
func (o *Octree) Clear() {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.root = &OctreeNode{
		Center:   o.root.Center,
		HalfSize: o.root.HalfSize,
		Objects:  make([]*TrackedObject, 0, maxObjectsPerNode),
		IsLeaf:   true,
	}
}

// Utility functions

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
