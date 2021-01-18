//
// Generated by sadl
//
package example.crudl;
import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.databind.annotation.JsonDeserialize;
import com.fasterxml.jackson.databind.annotation.JsonPOJOBuilder;
import com.fasterxml.jackson.databind.annotation.JsonSerialize;
import java.time.Instant;
import java.util.List;
import java.util.UUID;
import javax.validation.constraints.NotNull;

@JsonDeserialize(builder = NotFound.NotFoundBuilder.class)
@JsonIgnoreProperties({"message", "stackTrace", "cause", "localizedMessage", "suppressed"})
public class NotFound extends RuntimeException {
    @JsonInclude(JsonInclude.Include.NON_EMPTY) /* Optional field */
    private final String error;

    public NotFound(String error) {
        this.error = error;
    }

    public String getError() {
        return error;
    }

    public static NotFoundBuilder builder() {
        return new NotFoundBuilder();
    }

    @JsonPOJOBuilder(withPrefix="")
    public static class NotFoundBuilder {
        private String error;

        public NotFoundBuilder error(String error) {
            this.error = error;
            return this;
        }

        public NotFound build() {
            return new NotFound(error);
        }
    }
}