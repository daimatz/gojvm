import java.util.HashSet;
import java.util.ArrayList;
import java.util.Collections;

public class HashSetTest {
    public static void main(String[] args) {
        HashSet<String> set = new HashSet<>();
        set.add("apple");
        set.add("banana");
        set.add("cherry");
        set.add("apple"); // duplicate

        System.out.println(set.size());           // 3
        System.out.println(set.contains("banana")); // true
        System.out.println(set.contains("grape"));  // false

        set.remove("banana");
        System.out.println(set.size());           // 2

        // Sorted output for deterministic test
        ArrayList<String> sorted = new ArrayList<>(set);
        Collections.sort(sorted);
        for (int i = 0; i < sorted.size(); i++) {
            System.out.println(sorted.get(i));
        }
        // apple
        // cherry
    }
}
