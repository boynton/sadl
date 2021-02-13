namespace crudl

///
/// A CRUDL storage service as an example HTTP Web Service
///
service Crudl {
    version: "1",
    operations: [CreateItem, GetItem, PutItem, DeleteItem, ListItems]
}

///
/// Items use this restricted string as an identifier
///
@pattern("^[a-zA-Z_][a-zA-Z_0-9]*$")
string ItemId

///
/// The items to be stored.
///
structure Item {
  @required
  id: ItemId,
  modified: Timestamp,
  data: String,
}

///
/// A paginated list of items
///
structure ItemListing {
  @required
  items: ItemListingItems,
  next: ItemId,
}

///
/// If not modified, this is the response, with no content. "NotModified" is only used for the app
/// to throw the exception. i.e. in Java: throw new ServiceException(new NotModified())
///
@httpError(304)
@error("redirect")
structure NotModified {
  message: String,
}

@httpError(400)
@error("client")
structure BadRequest {
  message: String,
}

@httpError(404)
@error("client")
structure NotFound {
  message: String,
}

   
///
/// Create the item. The item with the updated modified time is returned.
///
@http(method: "POST", uri: "/items", code: 201)
operation CreateItem {
    input: CreateItemInput,
    output: CreateItemOutput,
    errors: [BadRequest],
}

structure CreateItemInput {
  @httpPayload
  item: Item,
}

structure CreateItemOutput {
  @httpPayload
  item: Item,
}

///
/// Get the item with the specified id. Conditional response is provided to avoid sending the item
/// over the wire if it has not changed.
///
@http(method: "GET", uri: "/items/{id}", code: 200)
@readonly
operation GetItem {
    input: GetItemInput,
    output: GetItemOutput,
    errors: [NotModified, NotFound],
}

structure GetItemInput {
  @httpLabel
  @required
  id: ItemId,
  @httpHeader("If-Modified-Since")
  ifNewer: Timestamp,
}

structure GetItemOutput {
  @httpPayload
  item: Item,
  @httpHeader("Modified")
  modified: Timestamp,
}

///
/// Update the item. The item with the updated modified time is returned.
///
@http(method: "PUT", uri: "/items/{id}", code: 200)
@idempotent
operation PutItem {
    input: PutItemInput,
    output: PutItemOutput,
    errors: [BadRequest],
}

structure PutItemInput {
  @required
  @httpLabel
  id: ItemId,
  @httpPayload
  item: Item,
}

structure PutItemOutput {
  @httpPayload
  item: Item,
}

///
/// Delete the item from the store.
///
@http(method: "DELETE", uri: "/items/{id}", code: 204)
@idempotent
operation DeleteItem {
    input: DeleteItemInput,
    errors: [NotFound],
}

structure DeleteItemInput {
  @httpLabel
  @required
  id: ItemId,
}

///
/// List the items. By default only 10 items are returned, but that can be overridden with a query
/// parameter. If more items are available than the limit, then a "next" token is returned, which
/// can be provided with a subsequent call as the "skip" query parameter.
///
@http(method: "GET", uri: "/items", code: 200)
@readonly
operation ListItems {
    input: ListItemsInput,
    output: ListItemsOutput,
}

structure ListItemsInput {
  @httpQuery("limit")
  limit: Integer,
  @httpQuery("skip")
  skip: ItemId,
}

structure ListItemsOutput {
  @httpPayload
  items: ItemListing,
}

list ItemListingItems {
    member: Item
}


