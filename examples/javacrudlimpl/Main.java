//
// Sample server main program and implementation of the crudl service
//
import org.eclipse.jetty.server.Server;
import org.glassfish.jersey.jetty.JettyHttpContainerFactory;
import org.glassfish.jersey.server.ResourceConfig;
import org.glassfish.hk2.utilities.binding.AbstractBinder;
import org.glassfish.jersey.jackson.JacksonFeature;
import javax.ws.rs.core.UriBuilder;
import java.io.IOException;
import java.net.URI;
import model.*;
import java.util.Map;
import java.util.HashMap;
import java.util.List;
import java.util.ArrayList;
import java.util.UUID;

public class Main {

    public static String BASE_URI = "http://localhost:8080/";

    public static void main(String[] args) {
        try {
            Server server = startServer(new CrudlImpl());
            server.join();
        } catch (Exception e) {
            System.err.println("*** " + e);
        }
    }

    public static Server startServer(CrudlImpl impl) throws Exception {
        URI baseUri = UriBuilder.fromUri(BASE_URI).build();
        ResourceConfig config = new ResourceConfig(CrudlResources.class);
        config.registerInstances(new AbstractBinder() {
                @Override
                protected void configure() {
                    bind(impl).to(CrudlService.class);
                }
            });
        Server server = JettyHttpContainerFactory.createServer(baseUri, config);
        server.start();
        System.out.println(String.format("Service started at %s", BASE_URI));
        return server;
    }


    //
    // Here is a memory-based implementation of the service
    //
    static class CrudlImpl implements CrudlService {

        Map<UUID,Item> storage = new HashMap<UUID,Item>();

        public CreateItemResponse CreateItem(CreateItemRequest req) {
            Item item = req.item;
            UUID key = item.id;
            synchronized (storage) {
                if (storage.containsKey(key)) {
                    throw new ServiceException("Already exists: " + key);
                }
                item.modified = Timestamp.now();
                storage.put(key, item);
            }
            return new CreateItemResponse().item(item);
        }

        public GetItemResponse GetItem(GetItemRequest req) {
            UUID key = req.id;
            synchronized (storage) {
                if (!storage.containsKey(key)) {
                    throw new ServiceException().entity(new NotFound().error("Item not found: " + key));
                }
                return new GetItemResponse().item(storage.get(key));
            }
        }

        public PutItemResponse PutItem(PutItemRequest req) {
            Item item = req.item;
            UUID key = item.id;
            synchronized (storage) {
                if (!storage.containsKey(key)) {
                    throw new ServiceException().entity(new NotFound().error("Item not found: " + key));
                }
                item.modified = Timestamp.now();
                storage.put(key, item);
                return new PutItemResponse().item(item);
            }
        }

        public DeleteItemResponse DeleteItem(DeleteItemRequest req) {
            UUID key = req.id;
            synchronized (storage) {
                if (!storage.containsKey(key)) {
                    throw new ServiceException().entity(new NotFound().error("Item not found: " + key));
                }
                storage.remove(key);
                return new DeleteItemResponse();
            }
        }

        public ListItemsResponse ListItems(ListItemsRequest req) {
            ListItemsResponse resp = new ListItemsResponse();
            List<Item> lst = new ArrayList<Item>();
            UUID next = null;
            int count = 0;
            int limit = req.limit;
            UUID skip = req.skip;
            for (Map.Entry<UUID,Item> e : storage.entrySet()) {
                UUID key = e.getKey();
                if (skip != null) {
                    if (!skip.equals(key)) {
                        continue;
                    }
                    skip = null;
                }
                count++;
                if (count > limit) {
                    next = e.getKey();
                    break;
                }
                lst.add(e.getValue());
            }
            return new ListItemsResponse().items(new ItemList().items(lst).next(next));
        }

    }
}
