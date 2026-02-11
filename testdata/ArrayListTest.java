import java.util.ArrayList;

public class ArrayListTest {
    public static void main(String[] args) {
        ArrayList<String> list = new ArrayList<>();
        list.add("Alice");
        list.add("Bob");
        list.add("Charlie");
        System.out.println(list.size());     // 3
        System.out.println(list.get(0));     // Alice
        System.out.println(list.get(2));     // Charlie
        list.set(1, "Beth");
        System.out.println(list.get(1));     // Beth

        // for-each with ArrayList (uses Iterator)
        for (String name : list) {
            System.out.println(name);
        }
    }
}
