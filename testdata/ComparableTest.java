import java.util.ArrayList;
import java.util.Collections;

public class ComparableTest {
    static class Student implements Comparable<Student> {
        String name;
        int score;
        Student(String name, int score) {
            this.name = name;
            this.score = score;
        }
        // Sort by score descending, then by name ascending
        public int compareTo(Student other) {
            if (this.score != other.score) {
                return other.score - this.score; // descending
            }
            return this.name.compareTo(other.name); // ascending
        }
    }

    public static void main(String[] args) {
        ArrayList<Student> students = new ArrayList<>();
        students.add(new Student("Charlie", 85));
        students.add(new Student("Alice", 95));
        students.add(new Student("Bob", 85));
        students.add(new Student("Diana", 90));

        Collections.sort(students);

        for (int i = 0; i < students.size(); i++) {
            Student s = students.get(i);
            System.out.println(s.name + ":" + s.score);
        }
        // Expected output (sorted by score desc, name asc):
        // Alice:95
        // Diana:90
        // Bob:85
        // Charlie:85
    }
}
