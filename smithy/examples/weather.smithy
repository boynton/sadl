namespace example.weather

service Weather {
  version: "2006-03-01",
  resources: [City],
  operations: [GetCurrentTime]
}

resource City {
  identifiers: { cityId: CityId },
  read: GetCity,
  list: ListCities,
  resources: [Forecast]
}

resource Forecast {
  type: resource,
  identifiers: { cityId: CityId },
  read: GetForecast,
}

// "pattern" is a trait
@pattern("^A-Za-z0-9 ]+$")
string CityId

@readonly
operation GetCity(GetCityInput) -> GetCityOutput

structure GetCityInput {
  // "cityId" provides the identifier for the resource and
  // has to be marked as required
  @required
  cityId: CityId
}

structure GetCityOutput {
  // "required" is used on output to indicate if the service
  // will always provide a value for the member
  @required
  name: smithy.api#String,

  @required
  coordinates: CityCoordinates,
}

structure CityCoordinates {
  @required
  latitude: smithy.api#Float,

  @required
  longitude: smithy.api#Float
}

structure NoSuchResource {
  @required
  resourceType: smithy.api#String
}
