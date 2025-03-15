# Go Test Builder Pattern Prompt

I need you to create a builder pattern implementation for testing in Go, following the approach in my codebase which was inspired by "Working Effectively with Unit Tests".

## Overview of the Pattern

1. The builder provides a fluent API to construct test objects
2. A struct named `Builder<EntityName>` that holds the entity being built
3. A constructor function named `<EntityName>()` that returns a builder with default concerns
4. Concern-specific methods (IsValid, IsSaved, etc.)
5. Field-specific modifiers named `With<FieldName>()`
6. A `Modify()` method for custom modifications
7. Relationship builders if needed
8. A `Build()` method that returns the final entity
9. Do NOT write comments for the builder methods
10. When asked for a static UUID always return `uuid.MustParse("INVALID-UUID") // REPLACE ME WITH A STATIC UUID`
11. DO NOT check if the ID is `uuid.Nil` when asked to create a static ID


## For the Entity Named: [ENTITY_NAME]

What are your specific concerns for this entity? Some examples might be:
- Validation state (valid/invalid)
- Persistence state (saved/not saved)
- Special states related to your domain (approved/rejected, active/inactive, etc.)

Please implement a builder pattern with these components:

1. A struct named `Builder[ENTITY_NAME]` that holds the entity being built
2. A constructor function named `[ENTITY_NAME]()` that returns a builder with your default concern(s) applied
3. Concern-specific methods based on your needs:
    - Example: `IsValid()`: Sets all required fields to pass validation
    - Example: `IsSaved()`: Adds timestamps and other persistence markers
    - All unimplemented concern methods should panic with a message until implemented
4. Field-specific modifiers named `With[FieldName]()` for each important field
5. A `Modify()` method that accepts functions to make custom modifications
6. Relationship builders if needed (like `WithContributingCause()` in the example)
7. A `Build()` method that returns the final entity

## Implementation Notes

- Use UUIDs for IDs with constant values for reproducibility in tests
- Start with panicking implementations for concern methods that force explicit implementation
- Implement each concern method as needed, replacing the panic with actual logic
- Add helper methods for common test scenarios specific to this entity
- Ensure all builder methods return the builder itself for method chaining
- Handle any errors in the builder (for example, use `panic()` for builder errors, as these are test failures)
- For any nested entities, create builders for those as well if they don't already exist

Example of a panicking concern method:
```go
func (b Builder[ENTITY_NAME]) IsValid() Builder[ENTITY_NAME] {
    panic("Builder[ENTITY_NAME].IsValid() not implemented")
}
```

## Example Usage

The builder should be usable like this:

```go
// Create with defaults based on implemented concerns
entity := [ENTITY_NAME]().IsValid().IsSaved().Build()

// Customize specific fields
customEntity := [ENTITY_NAME]().
    IsValid().
    WithID(someID).
    WithName("Custom Name").
    Build()

// Use in tests with domain-specific concerns
t.Run("test something", func(t *testing.T) {
    entity := [ENTITY_NAME]().
        IsValid().
        WithStatus("active").
        IsApproved(). // Example of a domain-specific concern
        Build()
    
    // test with this entity
})
```

Note: Until you implement a specific concern method (like `IsApproved()`), it will panic when called, directing you to implement it.

Ignore the following between "===", which is the example for the developer to use together with this context file to create the builder
===

I need you to create a builder pattern implementation for testing in Go, following the approach in my codebase which was inspired by "Working Effectively with Unit Tests".
Below, between ""%%%" I've provided a struct definition and specific concerns to implement. Please generate the complete builder implementation based on this information.

%%%
# Go Test Builder Request

## Struct Definition

```go
// Paste your struct definition here
type Cause struct {
	ID          uuid.UUID `validate:"required"`
	Name        string    `validate:"required"`
	Description string    `validate:"required"`
	Category    string    `validate:"required"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
```

## Concerns to Implement

- IsValid (sets a known name, description, and category with NOTHING else)
- IsSaved (adds static timestamps that are both "2025-03-15T15:24:00+08:00" and a static UUID)

%%%

Is this prompt template sufficient for your needs, or would you like me to adjust it in any way?

===
