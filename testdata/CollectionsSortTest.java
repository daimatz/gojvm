import java.util.ArrayList;
import java.util.Collections;

public class CollectionsSortTest {
    public static void main(String[] args) {
        ArrayList<String> list = new ArrayList<>();
        list.add("Charlie");
        list.add("Alice");
        list.add("Bob");
        Collections.sort(list);
        for (String s : list) {
            System.out.println(s);
        }

        ArrayList<Integer> nums = new ArrayList<>();
        nums.add(30);
        nums.add(10);
        nums.add(20);
        Collections.sort(nums);
        for (int n : nums) {
            System.out.println(n);
        }
    }
}
