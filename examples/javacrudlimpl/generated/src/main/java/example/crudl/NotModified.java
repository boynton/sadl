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

//If not modified, this is the response, with no content. "NotModified" is only used for the app to
//throw the exception. i.e. in Java: throw new ServiceException(new NotModified())
@JsonDeserialize(builder = NotModified.NotModifiedBuilder.class)
@JsonIgnoreProperties({})
public class NotModified extends RuntimeException {
    public NotModified() {
    }

    public static NotModifiedBuilder builder() {
        return new NotModifiedBuilder();
    }

    @JsonPOJOBuilder(withPrefix="")
    public static class NotModifiedBuilder {

        public NotModified build() {
            return new NotModified();
        }
    }
}
