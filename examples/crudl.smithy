namespace example

///
/// A CRUDL storage service as an example HTTP Web Service
///
service crudl {
  version: "1",
  resources: [ItemResource]
}

// If I call this "Item", I cannot define a structure called Item. Smithy misfeature!
// SADL current parses this and ignores it anyway, as it is not needed for code generation.
resource ItemResource {
  identifiers: {
    id: ItemId,
  },
  create: CreateItem,
  read: GetItem,
  update: PutItem,
  delete: DeleteItem,
  list: ListItems,
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

  /// The id is always provided by the client
  @required
  id: ItemId,

  /// The modified time is managed by the server
  modified: Timestamp,

  /// Other fields like this are not used by the server, but preserved.
  data: String
}

@readonly
@http(method: "GET", uri: "/{id}", code: 200)
operation GetItem {
  input: GetItemInput,
  output: GetItemOutput,
  errors: [NotModified, NotFound]
}

structure GetItemInput {
  @required
  @httpLabel
  id: ItemId,
  
  @httpHeader("If-Modified-Since")
  ifNewer: Timestamp,
}

structure GetItemOutput {
  @required
  @httpPayload
  item: Item,

  @httpHeader("Modified")
  modified: Timestamp,
}

@readonly
@http(method: "GET", uri: "/items", code: 200)
operation ListItems {
  input: ListItemsInput,
  output: ListItemsOutput,
  errors: [BadRequest],
}

structure ListItemsInput {
  @httpQuery("limit")
  limit: Integer,
  @httpQuery("skip")
  skip: ItemId,
}

structure ListItemsOutput {
  @required
  listings: ItemListing
}
structure ItemListing {
  @required
  items: Items,
  next: ItemId,
}

list ItemListingItems {
    member: Item
}

@error("client")
@httpError(304)
structure NotModified {
  message: String
}

@error("client")
@httpError(404)
structure NotFound {
  message: String
}

@error("client")
@httpError(400)
structure BadRequest {
  message: String
}

