import "@typespec/http";
import "@typespec/json-schema";
import "@typespec/openapi3";

import "./models.tsp";
import "./responses.tsp";
import "./public.tsp";
import "./admin.tsp";

using Http;

@service(#{ title: "Call Calendar API" })
@route("/api")
namespace CallCalendar;

/** Reports whether the API and its PostgreSQL connection are available. */
@tag("Health")
@route("/health")
interface HealthOperations {
  @get
  getHealth(): HealthOkResponse | HealthUnavailableResponse;
}
