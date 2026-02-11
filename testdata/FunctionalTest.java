import java.util.ArrayList;

public class FunctionalTest {
    // Custom functional interfaces (like java.util.function)
    interface Predicate {
        boolean test(int n);
    }

    interface Transform {
        int apply(int n);
    }

    static ArrayList<Integer> filter(ArrayList<Integer> list, Predicate p) {
        ArrayList<Integer> result = new ArrayList<>();
        for (int i = 0; i < list.size(); i++) {
            int val = (Integer) list.get(i);
            if (p.test(val)) {
                result.add(val);
            }
        }
        return result;
    }

    static ArrayList<Integer> map(ArrayList<Integer> list, Transform t) {
        ArrayList<Integer> result = new ArrayList<>();
        for (int i = 0; i < list.size(); i++) {
            int val = (Integer) list.get(i);
            result.add(t.apply(val));
        }
        return result;
    }

    static int reduce(ArrayList<Integer> list, int init, Transform acc) {
        // Using Transform creatively: acc.apply(init + val)
        int result = init;
        for (int i = 0; i < list.size(); i++) {
            result += (Integer) list.get(i);
        }
        return result;
    }

    static void printList(ArrayList<Integer> list) {
        StringBuilder sb = new StringBuilder();
        for (int i = 0; i < list.size(); i++) {
            if (i > 0) sb.append(",");
            sb.append(list.get(i));
        }
        System.out.println(sb.toString());
    }

    public static void main(String[] args) {
        ArrayList<Integer> nums = new ArrayList<>();
        for (int i = 1; i <= 10; i++) {
            nums.add(i);
        }

        // Filter even numbers (lambda)
        ArrayList<Integer> evens = filter(nums, n -> n % 2 == 0);
        printList(evens); // 2,4,6,8,10

        // Map: square each number
        ArrayList<Integer> squares = map(evens, n -> n * n);
        printList(squares); // 4,16,36,64,100

        // Reduce: sum
        int sum = reduce(squares, 0, n -> n);
        System.out.println(sum); // 220

        // Chained operations
        ArrayList<Integer> result = filter(nums, n -> n > 5);
        result = map(result, n -> n * 2);
        printList(result); // 12,14,16,18,20
    }
}
