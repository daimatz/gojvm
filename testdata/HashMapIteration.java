import java.util.HashMap;
import java.util.TreeMap;

public class HashMapIteration {
    public static void main(String[] args) {
        HashMap<String, Integer> map = new HashMap<>();
        map.put("Alice", 90);
        map.put("Bob", 85);
        map.put("Charlie", 95);

        System.out.println(map.size());         // 3
        System.out.println(map.get("Bob"));     // 85
        System.out.println(map.containsKey("Alice")); // true
        System.out.println(map.containsKey("Dave"));  // false

        // Use TreeMap for sorted iteration (avoids Collections.sort)
        TreeMap<String, Integer> sorted = new TreeMap<>(map);
        for (String key : sorted.keySet()) {
            System.out.println(key + "=" + sorted.get(key));
        }
    }
}
