//
// Generated by sadl
//
package example.crudl;

import java.util.*;
import java.time.Instant;
import javax.inject.Inject;
import javax.ws.rs.*;
import javax.ws.rs.core.*;
import static javax.ws.rs.core.Response.Status;
import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonInclude.Include;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializationFeature;
import com.fasterxml.jackson.databind.DeserializationFeature;

@Path("/")
public class CrudlResources {

    @Inject
    private Crudl crudlController;

    
    @POST
    @Path("/items")
    @Produces(MediaType.APPLICATION_JSON)
    public Response createItem(Item item) {
        CreateItemRequest req = CreateItemRequest.builder().item(item).build();
        try {
            CreateItemResponse res = crudlController.createItem(req);
            return Response.status(201).entity(res.getItem()).build();
        } catch (BadRequest e) {
                throw new WebApplicationException(Response.status(400).entity(e).build());
        } catch (Throwable th) {
            return Response.status(500).build();
        }
    }

    
    @GET
    @Path("/items/{id}")
    @Produces(MediaType.APPLICATION_JSON)
    public Response getItem(@PathParam("id") UUID id, @HeaderParam("If-Modified-Since") Instant ifNewer) {
        GetItemRequest req = GetItemRequest.builder().id(id).ifNewer(ifNewer).build();
        try {
            GetItemResponse res = crudlController.getItem(req);
            return Response.status(200).header("Modified", res.getModified()).entity(res.getItem()).build();
        } catch (NotModified e) {
                throw new WebApplicationException(Response.status(304).entity(e).build());
        } catch (NotFound e) {
                throw new WebApplicationException(Response.status(404).entity(e).build());
        } catch (Throwable th) {
            return Response.status(500).build();
        }
    }

    
    @PUT
    @Path("/items/{id}")
    @Produces(MediaType.APPLICATION_JSON)
    public Response putItem(@PathParam("id") UUID id, Item item) {
        PutItemRequest req = PutItemRequest.builder().id(id).item(item).build();
        try {
            PutItemResponse res = crudlController.putItem(req);
            return Response.status(200).entity(res.getItem()).build();
        } catch (BadRequest e) {
                throw new WebApplicationException(Response.status(400).entity(e).build());
        } catch (Throwable th) {
            return Response.status(500).build();
        }
    }

    
    @DELETE
    @Path("/items/{id}")
    @Produces(MediaType.APPLICATION_JSON)
    public Response deleteItem(@PathParam("id") UUID id) {
        DeleteItemRequest req = DeleteItemRequest.builder().id(id).build();
        try {
            crudlController.deleteItem(req);
            return Response.status(204).build();
        } catch (NotFound e) {
                throw new WebApplicationException(Response.status(404).entity(e).build());
        } catch (Throwable th) {
            return Response.status(500).build();
        }
    }

    
    @GET
    @Path("/items")
    @Produces(MediaType.APPLICATION_JSON)
    public Response listItems(@DefaultValue("10") @QueryParam("limit") Integer limit, @QueryParam("skip") UUID skip) {
        ListItemsRequest req = ListItemsRequest.builder().limit(limit).skip(skip).build();
        try {
            ListItemsResponse res = crudlController.listItems(req);
            return Response.status(200).entity(res.getItems()).build();
        } catch (Throwable th) {
            return Response.status(500).build();
        }
    }



}